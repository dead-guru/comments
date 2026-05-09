// @ts-check
const config = {
  title: "Deadcomments Docs Demo",
  tagline: "A local stand for testing embedded comments",
  favicon: "img/favicon.ico",
  url: "http://localhost:3000",
  baseUrl: "/",
  organizationName: "deadcomments",
  projectName: "deadcomments",
  onBrokenLinks: "throw",
  markdown: {
    hooks: {
      onBrokenMarkdownLinks: "warn"
    }
  },
  scripts: [{src: "/deadcomments-loader.js", defer: true}],
  presets: [
    [
      "classic",
      {
        docs: {
          sidebarPath: require.resolve("./sidebars.js"),
          routeBasePath: "/docs"
        },
        blog: false,
        theme: {
          customCss: require.resolve("./src/css/custom.css")
        }
      }
    ]
  ],
  themeConfig: {
    navbar: {
      title: "Deadcomments Demo",
      items: [{type: "docSidebar", sidebarId: "tutorialSidebar", position: "left", label: "Docs"}]
    }
  }
};

module.exports = config;
