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
                
                formatted_tools.append({
                    "type": "function",
                    "function": {
                        "name": name,
                        "description": description,
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
    
    while True:
        response_json = client.chat(messages, tools)
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
                tool_response = json.dumps(tool_result) if isinstance(tool_result, (dict, list)) else str(tool_result)
            except Exception as e:
                tool_response = f"Error executing tool {tool_name}: {str(e)}"
            
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
