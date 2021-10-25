import { defineClientAppEnhance } from '@vuepress/client'

const enGuide = {path: '/', redirect: '/guide/'}
const zhGuide = {path: '/zh', redirect: '/zh/guide/'}

export default defineClientAppEnhance(({ app, router, siteData }) => {
  router.addRoute(enGuide)
  router.addRoute(zhGuide)
})
