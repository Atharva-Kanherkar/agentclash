import { Template } from 'e2b'

export const template = Template()
  .fromImage('ubuntu:24.04')
  .setUser('root')
  .setWorkdir('/')

  // System essentials
  .runCmd(
    'apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends ' +
      'bash ca-certificates coreutils curl wget git software-properties-common gnupg ' +
      '&& rm -rf /var/lib/apt/lists/*',
  )

  // Node.js 20 LTS via NodeSource
  .runCmd(
    'curl -fsSL https://deb.nodesource.com/setup_20.x | bash - ' +
      '&& apt-get install -y nodejs ' +
      '&& rm -rf /var/lib/apt/lists/*',
  )

  // Python 3, Go, C/C++ toolchain
  .runCmd(
    'apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends ' +
      'python3 python3-pip python3-venv golang-go gcc g++ make cmake ' +
      '&& rm -rf /var/lib/apt/lists/*',
  )

  // Document processing
  .runCmd(
    'apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends ' +
      'poppler-utils pandoc ' +
      '&& rm -rf /var/lib/apt/lists/*',
  )

  // Data tools
  .runCmd(
    'apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends ' +
      'jq csvtool libxml2-utils sqlite3 ' +
      '&& rm -rf /var/lib/apt/lists/*',
  )

  // Search & text
  .runCmd(
    'apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends ' +
      'ripgrep fd-find ' +
      '&& rm -rf /var/lib/apt/lists/*',
  )

  // Python packages
  .runCmd(
    'pip3 install --no-cache-dir --break-system-packages ' +
      'PyPDF2 pdfplumber requests httpx csvkit',
  )

  // Helper scripts
  .copy('tools/', '/tools/', { user: 'root', mode: 0o755 })

  // Workspace setup
  .runCmd('mkdir -p /workspace/agentclash')
  .setWorkdir('/workspace')
  .setStartCmd('sleep infinity', 'sleep 20')
