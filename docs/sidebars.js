// @ts-check

/** @type {import('@docusaurus/plugin-content-docs').SidebarsConfig} */
const sidebars = {
  docsSidebar: [
    'quickstart',
    {
      type: 'category',
      label: 'Guide',
      collapsed: false,
      items: [
        'guide/install',
        'guide/helm-values',
        'guide/create-cluster',
        'guide/manage-cluster',
        'guide/storage',
        'guide/monitoring',
        'guide/access-control',
        'guide/networking',
        'guide/advanced-configuration',
        'guide/templates',
        'guide/operations',
        'guide/cluster-manager-ui',
        'guide/glossary',
        'guide/troubleshooting',
      ],
    },
    {
      type: 'category',
      label: 'API Reference',
      items: [
        'api-reference/aerospikecluster',
        'api-reference/aerospikeclustertemplate',
      ],
    },
  ],
};

export default sidebars;
