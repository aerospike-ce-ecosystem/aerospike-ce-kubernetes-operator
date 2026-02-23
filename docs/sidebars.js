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
        'guide/access-control',
      ],
    },
    {
      type: 'category',
      label: 'API Reference',
      items: [
        'api-reference/aerospikececluster',
      ],
    },
  ],
};

export default sidebars;
