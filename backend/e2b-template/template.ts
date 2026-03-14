import { Template } from 'e2b'

export const template = Template()
  .fromImage('ubuntu:24.04')
  .setUser('root')
  .setWorkdir('/')
  .runCmd('apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends bash ca-certificates coreutils && rm -rf /var/lib/apt/lists/*')
  .runCmd('id -u user >/dev/null 2>&1 || useradd -ms /bin/bash user')
  .runCmd('mkdir -p /workspace/agentclash')
  .runCmd('chown -R user:user /workspace')
  .setWorkdir('/workspace')
  .setUser('user')
  .setStartCmd('sleep infinity', 'sleep 20')
