/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
export const UNIROUTERS_ORIGIN = 'https://unirouters.cc'
export const UNIROUTERS_API_BASE_URL = `${UNIROUTERS_ORIGIN}/v1`
export const UNIROUTERS_DOCS_PATH = '/docs'
export const UNIROUTERS_API_KEY_PLACEHOLDER = 'sk-unirouters-your-key'
export const UNIROUTERS_DEFAULT_MODEL = 'gpt-5.6-sol'
export const UNIROUTERS_CLAUDE_MODEL = 'claude-opus-5'
export const CODEX_INSTALL_URL = 'https://developers.openai.com/codex/cli'
export const CLAUDE_CODE_INSTALL_URL =
  'https://code.claude.com/docs/en/getting-started'

export const DOCS_SECTIONS = [
  { id: 'overview', labelKey: 'docs.nav.overview' },
  { id: 'quick-start', labelKey: 'docs.nav.quickStart' },
  { id: 'authentication', labelKey: 'docs.nav.authentication' },
  { id: 'openai-sdks', labelKey: 'docs.nav.openaiSdks' },
  { id: 'coding-agents', labelKey: 'docs.nav.codingAgents' },
  { id: 'api-reference', labelKey: 'docs.nav.apiReference' },
  { id: 'troubleshooting', labelKey: 'docs.nav.troubleshooting' },
] as const

export const RESPONSE_CURL_EXAMPLE = `export UNIROUTERS_API_KEY="${UNIROUTERS_API_KEY_PLACEHOLDER}"

curl ${UNIROUTERS_API_BASE_URL}/responses \\
  -H "Authorization: Bearer $UNIROUTERS_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "${UNIROUTERS_DEFAULT_MODEL}",
    "input": "Review this repository and propose the safest refactor.",
    "stream": true
  }'`

export const PYTHON_SDK_EXAMPLE = `from openai import OpenAI
import os

client = OpenAI(
    base_url="${UNIROUTERS_API_BASE_URL}",
    api_key=os.environ["UNIROUTERS_API_KEY"],
)

response = client.responses.create(
    model="${UNIROUTERS_DEFAULT_MODEL}",
    input="Explain the architecture of this project.",
)

print(response.output_text)`

export const TYPESCRIPT_SDK_EXAMPLE = `import OpenAI from "openai";

const client = new OpenAI({
  baseURL: "${UNIROUTERS_API_BASE_URL}",
  apiKey: process.env.UNIROUTERS_API_KEY,
});

const response = await client.responses.create({
  model: "${UNIROUTERS_DEFAULT_MODEL}",
  input: "Find the highest-risk code path in this project.",
});

console.log(response.output_text);`

export const CODEX_KEY_EXAMPLE = `export UNIROUTERS_API_KEY="${UNIROUTERS_API_KEY_PLACEHOLDER}"

codex`

export const CODEX_CONFIG_EXAMPLE = `model = "${UNIROUTERS_DEFAULT_MODEL}"
model_provider = "unirouters"

[model_providers.unirouters]
name = "uniRouters"
base_url = "${UNIROUTERS_API_BASE_URL}"
env_key = "UNIROUTERS_API_KEY"
wire_api = "responses"`

export const CLAUDE_CODE_EXAMPLE = `export ANTHROPIC_BASE_URL="${UNIROUTERS_ORIGIN}"
export ANTHROPIC_AUTH_TOKEN="${UNIROUTERS_API_KEY_PLACEHOLDER}"
export ANTHROPIC_MODEL="${UNIROUTERS_CLAUDE_MODEL}"

claude`
