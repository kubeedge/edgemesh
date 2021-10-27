import type { SidebarConfig } from '@vuepress/theme-default'

const guide = {
  text: 'Guide',
  children: [
    '/guide/README.md',
    '/guide/getting-started.md',
    '/guide/test-case.md',
    '/guide/edge-gateway.md',
  ],
}

export const en: SidebarConfig = {
  '/': [
    guide,
  ],
  '/guide/': [
    guide,
  ],
  '/advanced/': [
    {
      text: 'Advanced',
      children: [
        '/advanced/architecture.md',
        '/advanced/hybirdnat.md',
        '/advanced/security.md',
      ],
    },
  ],
}
