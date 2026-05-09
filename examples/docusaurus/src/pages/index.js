import React from "react";
import Link from "@docusaurus/Link";
import Layout from "@theme/Layout";

export default function Home() {
  return (
    <Layout title="Deadcomments Demo">
      <main className="container margin-vert--lg">
        <h1>Deadcomments Docusaurus Demo</h1>
        <p>Open the docs pages to test embedded comments from the local server.</p>
        <Link className="button button--primary" to="/docs/intro">
          Open demo docs
        </Link>
      </main>
    </Layout>
  );
}
