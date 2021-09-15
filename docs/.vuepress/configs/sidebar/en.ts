import type { SidebarConfig } from '@vuepress/theme-default'

export const en: SidebarConfig = {
  '/guide/': [
    {
      text: 'Guide',
      children: [
        '/guide/README.md',
        '/guide/getting-started.md',
        '/guide/test-case.md',
        '/guide/edge-gateway.md',
      ],
    },
  ],
  '/advanced': [
    {
      text: 'Advanced',
      children: [
        '/advanced/architecture.md',
        '/advanced/hybirdnat.md',
      ],
    },
  ],
}
