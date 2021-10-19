import type { NavbarConfig } from '@vuepress/theme-default'

export const zh: NavbarConfig = [
  {
    text: '指南',
    link: '/zh/guide/',
  },
  {
    text: '了解更多',
    children: [
      {
        text: '深入',
        children: [
          '/zh/advanced/architecture.md',
          '/zh/advanced/hybirdnat.md',
          '/zh/advanced/security.md',
        ],
      }
    ],
  },
  {
    text: '版本',
    children: [
      {
        text: '更新日志',
        link:
          'https://github.com/kubeedge/kubeedge/blob/master/CHANGELOG/README.md',
      },
      {
        text: 'v1.8',
        link: 'https://edgemesh.netlify.app/zh/',
      },
    ],
  },
]
