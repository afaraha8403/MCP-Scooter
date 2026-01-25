import os
import json
import time
import requests
from typing import Any, List, Dict, Tuple

class OpenRouterClient:
    def __init__(self, api_key: str, model: str):
        self.api_key = api_key
        self.model = model
        self.base_url = "https://openrouter.ai/api/v1/chat/completions"
        self.headers = {
            "Authorization": f"Bearer {self.api_key}",
            "HTTP-Referer": "https://github.com/mcp-scooter/scooter",
            "X-Title": "MCP Scooter Evaluation",
            "Content-Type": "application/json"
        }

    def chat(self, messages: List[Dict[str, Any]], tools: List[Dict[str, Any]] = None) -> Dict[str, Any]:
        payload = {
            "model": self.model,
            "messages": messages,
        }
        if tools:
            # Convert MCP tools to OpenAI/OpenRouter tool format
            formatted_tools = []
            for tool in tools:
                # Handle both MCP Tool object and direct dict
                input_schema = getattr(tool, "inputSchema", tool.get("inputSchema"))
                name = getattr(tool, "name", tool.get("name"))
                description = getattr(tool, "description", tool.get("description"))
                sample_input = getattr(tool, "sampleInput", tool.get("sampleInput"))
                
                # Ensure input_schema is a dict and has required fields for OpenAI format
                if input_schema is None:
                    input_schema = {"type": "object", "properties": {}}
                
                # If input_schema is a JSONSchema object, get its dict representation
                if hasattr(input_schema, "to_dict"):
                    input_schema = input_schema.to_dict()
                
                # Final safety check: if it's still not a dict, force it to be one
                if not isinstance(input_schema, dict):
                    input_schema = {"type": "object", "properties": {}}

                # Enhance description with sample input if available
                enhanced_description = description or ""
                if sample_input:
                    if isinstance(sample_input, dict):
                        sample_json = json.dumps(sample_input)
                    else:
                        sample_json = str(sample_input)
                    enhanced_description += f"\n\nExample usage: {sample_json}"

                formatted_tools.append({
                    "type": "function",
                    "function": {
                        "name": name,
                        "description": enhanced_description,
                        "parameters": input_schema
                    }
                })
            payload["tools"] = formatted_tools

        response = requests.post(self.base_url, headers=self.headers, json=payload)
        if response.status_code != 200:
            raise Exception(f"OpenRouter API Error: {response.status_code} - {response.text}")
        
        return response.json()

async def agent_loop_openrouter(
    client: OpenRouterClient,
    question: str,
    tools: List[Dict[str, Any]],
    connection: Any,
    system_prompt: str
) -> Tuple[str, Dict[str, Any]]:
    """Run the agent loop using OpenRouter directly."""
    messages = [
        {"role": "system", "content": system_prompt},
        {"role": "user", "content": question}
    ]
    
    tool_metrics = {}
    # Keep a mutable reference to tools so we can refresh after activation
    current_tools = list(tools)
    
    while True:
        response_json = client.chat(messages, current_tools)
        choice = response_json["choices"][0]
        message = choice["message"]
        
        messages.append(message)
        
        if choice["finish_reason"] != "tool_calls":
            return message.get("content", ""), tool_metrics
            
        tool_calls = message.get("tool_calls", [])
        for tool_call in tool_calls:
            tool_name = tool_call["function"]["name"]
            tool_args = json.loads(tool_call["function"]["arguments"])
            
            tool_start_ts = time.time()
            try:
                print(f"  🛠️ Calling tool: {tool_name}")
                tool_result = await connection.call_tool(tool_name, tool_args)
                
                # The tool_result from mcp-builder connection might be a complex object
                # or a list of TextContent objects. We need to extract the actual content.
                
                if hasattr(tool_result, "content"):
                    # It's an MCP response object
                    content_list = []
                    for item in tool_result.content:
                        if hasattr(item, "text"):
                            content_list.append(item.text)
                        elif isinstance(item, dict) and "text" in item:
                            content_list.append(item["text"])
                        else:
                            content_list.append(str(item))
                    tool_response = "\n".join(content_list)
                elif isinstance(tool_result, list):
                    # It might be a list of TextContent objects
                    content_list = []
                    for item in tool_result:
                        if hasattr(item, "text"):
                            content_list.append(item.text)
                        elif isinstance(item, dict) and "text" in item:
                            content_list.append(item["text"])
                        else:
                            content_list.append(str(item))
                    tool_response = "\n".join(content_list)
                elif isinstance(tool_result, (dict, list)):
                    tool_response = json.dumps(tool_result)
                else:
                    tool_response = str(tool_result)
                
                print(f"  ✅ Tool {tool_name} returned: {tool_response[:100]}...")
            except Exception as e:
                tool_response = f"Error executing tool {tool_name}: {str(e)}"
                print(f"  ❌ Tool {tool_name} failed: {tool_response}")
            
            tool_duration = time.time() - tool_start_ts
            
            if tool_name not in tool_metrics:
                tool_metrics[tool_name] = {"count": 0, "durations": []}
            tool_metrics[tool_name]["count"] += 1
            tool_metrics[tool_name]["durations"].append(tool_duration)
            
            messages.append({
                "role": "tool",
                "tool_call_id": tool_call["id"],
                "name": tool_name,
                "content": tool_response
            })
            
            # Refresh tools after activation so new tools become available to the LLM
            if tool_name in ("scooter_activate", "scooter_add", "scooter_deactivate"):
                try:
                    refreshed_tools = await connection.list_tools()
                    current_tools = list(refreshed_tools)
                    print(f"  🔄 Refreshed tool list: {len(current_tools)} tools available")
                except Exception as e:
                    print(f"  ⚠️ Failed to refresh tools: {e}")
