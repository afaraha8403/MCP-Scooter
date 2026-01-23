import os
import sys
import asyncio
import argparse
import yaml
import json
import time
import re
from datetime import datetime
from pathlib import Path
from typing import Any, List, Dict, Tuple
from openrouter_client import OpenRouterClient, agent_loop_openrouter

# Add the mcp-builder scripts to path
sys.path.append(str(Path(__file__).parent.parent.parent / ".agent" / "skills" / "mcp-builder" / "scripts"))

from connections import create_connection

EVALUATION_PROMPT = """You are an AI assistant with access to tools via the MCP Scooter gateway.

Your primary goal is to solve the user's request using the available tools.

## TOOL DISCOVERY WORKFLOW

1. **Check active tools first**: Use `scooter_list_active` to see what's already available.
2. **Search for tools**: Use `scooter_find` with a query like "search" or "github" to discover tools.
3. **Add tools**: Use `scooter_add` with the SERVER NAME (e.g., "brave-search", NOT "brave_web_search").
4. **Use tools**: Call the tool functions with the EXACT argument names from their schemas.

## CRITICAL: TOOL ARGUMENT FORMATS

When calling tools, you MUST use the exact argument names. Here are examples:

### brave_web_search (from brave-search server)
```json
{"query": "your search query here", "count": 5}
```
- Required: `query` (string) - The search query
- Optional: `count` (integer, 1-20) - Number of results

### search_repositories (from github server)  
```json
{"query": "mcp-server language:typescript"}
```
- Required: `query` (string) - GitHub search query

### scooter_find
```json
{"query": "search"}
```
- Optional: `query` (string) - Search term to filter tools

### scooter_add
```json
{"tool_name": "brave-search"}
```
- Required: `tool_name` (string) - The SERVER name, not the function name

## ERROR RECOVERY

If a tool returns "Invalid arguments":
1. Check the tool's inputSchema for required parameters
2. Verify you're using the exact parameter names (case-sensitive)
3. Ensure required parameters are provided
4. Try with minimal required arguments first

## OUTPUT FORMAT

When given a task, you MUST:
1. Use the available tools to complete the task
2. Provide summary of each step in your approach, wrapped in <summary> tags
3. Provide feedback on the tools provided, wrapped in <feedback> tags
4. Provide your final response, wrapped in <response> tags

Summary Requirements:
- In your <summary> tags, explain the steps you took, which tools you used, and why.

Feedback Requirements:
- In your <feedback> tags, provide constructive feedback on tool usability.

Response Requirements:
- Your response should be concise and directly address what was asked
- Always wrap your final response in <response> tags
- If you cannot solve the task return <response>NOT_FOUND</response>
- For names or text, provide the exact text requested
- Your response should go last"""

def extract_xml_content(text: str, tag: str) -> str | None:
    """Extract content from XML tags."""
    pattern = rf"<{tag}>(.*?)</{tag}>"
    matches = re.findall(pattern, text, re.DOTALL)
    return matches[-1].strip() if matches else None

async def evaluate_single_task(
    client: OpenRouterClient,
    qa_pair: dict[str, Any],
    tools: list[dict[str, Any]],
    connection: Any,
    task_index: int,
) -> dict[str, Any]:
    """Evaluate a single QA pair with the given tools."""
    start_time = time.time()

    print(f"Task {task_index + 1}: Running task with question: {qa_pair['question']}")
    response, tool_metrics = await agent_loop_openrouter(
        client, 
        qa_pair["question"], 
        tools, 
        connection,
        EVALUATION_PROMPT
    )

    response_value = extract_xml_content(response, "response")
    summary = extract_xml_content(response, "summary")
    feedback = extract_xml_content(response, "feedback")

    duration_seconds = time.time() - start_time

    return {
        "question": qa_pair["question"],
        "expected": qa_pair["answer"],
        "actual": response_value,
        "score": int(response_value == qa_pair["answer"]) if response_value else 0,
        "total_duration": duration_seconds,
        "tool_calls": tool_metrics,
        "num_tool_calls": sum(len(metrics["durations"]) for metrics in tool_metrics.values()),
        "summary": summary,
        "feedback": feedback,
    }

async def run_scenario_evaluation(scenarios_file, connection, model, results_dir):
    """Run multi-turn scenario evaluations."""
    with open(scenarios_file, 'r') as f:
        config = yaml.safe_load(f)
    
    scenarios = config.get('scenarios', [])
    print(f"📋 Loaded {len(scenarios)} multi-turn scenarios")
    
    api_key = os.getenv("OPENROUTER_API_KEY") or os.getenv("ANTHROPIC_API_KEY")
    client = OpenRouterClient(api_key=api_key, model=model)
    
    tools = await connection.list_tools()
    
    results = []
    for scenario in scenarios:
        print(f"🚀 Running Scenario: {scenario['name']}")
        
        # Get acceptable answers (can be a list)
        acceptable_answers = scenario.get('validation', {}).get('response_must_contain', [""])
        if not isinstance(acceptable_answers, list):
            acceptable_answers = [acceptable_answers]
        
        # Adapt scenario to the format expected by evaluate_single_task
        qa_pair = {
            "question": scenario['task'],
            "answer": acceptable_answers[0] if acceptable_answers else "",
            "acceptable_answers": acceptable_answers  # Pass all acceptable answers
        }
        
        result = await evaluate_single_task(client, qa_pair, tools, connection, 0)
        
        # Add scenario metadata to result
        result['scenario_id'] = scenario['id']
        result['name'] = scenario['name']
        result['acceptable_answers'] = acceptable_answers
        results.append(result)
        
    return results

async def main():
    parser = argparse.ArgumentParser(description="Run MCP Scooter Evaluation")
    parser.add_argument("--scooter-url", default=os.getenv("SCOOTER_URL", "http://127.0.0.1:6277"), help="Scooter Gateway URL")
    parser.add_argument("--profile", default="work", help="Scooter profile ID")
    parser.add_argument("--api-key", default=os.getenv("SCOOTER_API_KEY", ""), help="Scooter API Key")
    parser.add_argument("--output-dir", type=Path, default=Path(__file__).parent.parent / "results", help="Directory for results")
    parser.add_argument("--model", default="google/gemini-2.0-flash-001", help="Model name")
    parser.add_argument("--mode", choices=["qa", "scenarios"], default="scenarios", help="Evaluation mode")
    
    args = parser.parse_args()
    
    # Ensure results directory exists
    args.output_dir.mkdir(parents=True, exist_ok=True)
    
    # Scooter uses SSE for the gateway
    sse_url = f"{args.scooter_url}/profiles/{args.profile}/sse"
    
    headers = {}
    if args.api_key:
        headers["X-API-Key"] = args.api_key
        
    print(f"🔗 Connecting to Scooter at {sse_url}...")
    
    connection = create_connection(
        transport="sse",
        url=sse_url,
        headers=headers
    )
    
    async with connection:
        print("✅ Connected to Scooter")
        
        timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
        
        if args.mode == "qa":
            print("QA mode not implemented in direct OpenRouter version yet.")
        else:
            scenarios_file = Path(__file__).parent / "scenarios.yaml"
            results = await run_scenario_evaluation(scenarios_file, connection, args.model, args.output_dir)
            
            # Save raw results as JSON
            json_output = args.output_dir / f"scenario_results_{timestamp}.json"
            with open(json_output, 'w') as f:
                json.dump(results, f, indent=2)
            
            # Generate a simple markdown summary
            summary_md = f"# Scenario Evaluation Report - {timestamp}\n\n"
            passed_count = 0
            for r in results:
                # Check if any acceptable answer is in the response
                actual_lower = (r['actual'] or "").lower()
                acceptable_answers = r.get('acceptable_answers', [r['expected']])
                
                # Special case for empty expected (like Graceful Failure)
                if not acceptable_answers or acceptable_answers == [""]:
                    passed = True
                else:
                    passed = any(ans.lower() in actual_lower for ans in acceptable_answers if ans)
                
                if passed:
                    passed_count += 1
                
                status = "✅" if passed else "❌"
                summary_md += f"## {r['name']} {status}\n"
                summary_md += f"**Task**: {r['question']}\n\n"
                summary_md += f"**Summary**: {r['summary']}\n\n"
                summary_md += "---\n\n"
            
            # Add overall score
            total = len(results)
            summary_md += f"\n## Overall Score: {passed_count}/{total} ({100*passed_count//total}%)\n"
            
            report_file = args.output_dir / f"scenario_report_{timestamp}.md"
            report_file.write_text(summary_md, encoding="utf-8")
            print(f"✅ Scenario Report saved to {report_file}")
            print(f"✅ Raw results saved to {json_output}")

if __name__ == "__main__":
    asyncio.run(main())
