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
        'guide/create-cluster',
        'guide/manage-cluster',
        'guide/storage',
        'guide/access-control',
        'guide/cluster-templates',
        'guide/glossary',
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
