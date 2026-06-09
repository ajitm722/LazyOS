import type {SidebarsConfig} from '@docusaurus/plugin-content-docs';

const sidebars: SidebarsConfig = {
  docsSidebar: [
    {
      type: 'doc',
      id: 'intro',
      label: 'Introduction',
    },
    {
      type: 'doc',
      id: 'quick-start',
      label: 'Quick Start',
    },
    {
      type: 'doc',
      id: 'configuration',
      label: 'Configuration',
    },
    {
      type: 'category',
      label: 'Architecture',
      items: [
        'architecture/overview',
        'architecture/domain-model',
        'architecture/caching',
        'architecture/components',
      ],
    },
    {
      type: 'category',
      label: 'Development',
      items: [
        'development/unit-tests',
        'development/integration-tests',
        'development/building',
      ],
    },
    {
      type: 'category',
      label: 'Operations',
      items: [
        'operations/remote-deployment',
        'operations/mcp-server',
        {
          type: 'category',
          label: 'Diagnostics',
          items: [
            'operations/diagnostics/aws-bedrock',
            'operations/diagnostics/virtual-memory',
          ],
        },
      ],
    },
    {
      type: 'category',
      label: 'Reference',
      items: [
        'reference/ui-and-keys',
        'reference/osquery-interaction',
        'reference/sequence-flows',
      ],
    },
    {
      type: 'doc',
      id: 'api/index',
      label: 'API Reference',
    },
  ],
};

export default sidebars;
