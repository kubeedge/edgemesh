import type { NavbarConfig } from '@vuepress/theme-default'

export const en: NavbarConfig = [
  {
    text: 'Guide',
    link: '/guide/',
  },
  {
    text: 'Learn More',
    children: [
      {
        text: 'Advanced',
        children: [
          '/advanced/architecture.md',
        ],
      },
    ],
  },
  {
    text: 'Versions',
    children: [
      {
        text: 'CHANGELOG',
        link:
          'https://github.com/kubeedge/kubeedge/blob/master/CHANGELOG/README.md',
      },
      {
        text: 'v1.8',
        link: 'https://edgemesh.netlify.app/',
      },
    ],
  },
]
