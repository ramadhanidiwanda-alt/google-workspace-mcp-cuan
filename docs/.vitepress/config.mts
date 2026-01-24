import { defineConfig } from 'vitepress'

// https://vitepress.dev/reference/site-config
export default defineConfig({
  base: '/google-workspace-mcp/',
  title: "Google Workspace MCP",
  description: "MCP Server for Google Workspace APIs",
  themeConfig: {
    // https://vitepress.dev/reference/default-theme-config
    nav: [
      { text: 'Home', link: '/' },
      { text: 'Development', link: '/development' },
      { text: 'Release', link: '/release' },
      { text: 'Release Notes', link: '/release_notes' }
    ],

    sidebar: [
      {
        text: 'Documentation',
        items: [
          { text: 'Overview', link: '/' },
          { text: 'Development Guide', link: '/development' },
          { text: 'Release Guide', link: '/release' },
          { text: 'Release Notes', link: '/release_notes' }
        ]
      }
    ],

    socialLinks: [
      { icon: 'github', link: 'https://github.com/tomohiro-owada/google-workspace-mcp' }
    ]
  }
})
