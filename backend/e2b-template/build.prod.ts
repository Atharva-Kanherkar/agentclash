import { Template, defaultBuildLogger } from 'e2b'
import { template } from './template'

async function main() {
  await Template.build(template, 'agentclash-v1', {
    onBuildLogs: defaultBuildLogger(),
  });
}

main().catch(console.error);