import type { SidebarConfig } from '@vuepress/theme-default'

const guide = {
  text: 'Guide',
  children: [
    '/guide/README.md',
    '/guide/getting-started.md',
    '/guide/test-case.md',
    '/guide/edge-gateway.md',
     '/guide/security.md',
    '/guide/ssh.md',
  ],
}

const reference = {
  text: 'Reference',
  children: [
    '/reference/config-items.md',
  ],
}

const advanced = {
  text: 'Advanced',
  children: [
    '/advanced/architecture.md',
    '/advanced/hybirdnat.md',
  ],
}

export const en: SidebarConfig = {
  '/': [
    guide,
    reference,
  ],
  '/guide/': [
    guide,
    reference,
  ],
  '/advanced/': [
    advanced,
  ],
}
