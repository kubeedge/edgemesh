import type { SidebarConfig } from '@vuepress/theme-default'

const guide = {
  text: '指南',
  children: [
    '/zh/guide/README.md',
    '/zh/guide/getting-started.md',
    '/zh/guide/test-case.md',
    '/zh/guide/edge-gateway.md',
  ],
}

export const zh: SidebarConfig = {
  '/zh/': [
    guide,
  ],
  '/zh/guide/': [
    guide,
  ],
  '/zh/advanced/': [
    {
      text: '深入',
      children: [
        '/zh/advanced/architecture.md',
        '/zh/advanced/hybirdnat.md',
        '/zh/advanced/security.md',
      ],
    },
  ]
}
