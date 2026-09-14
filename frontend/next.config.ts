import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // Emits a minimal, self-contained server in .next/standalone: only the
  // files actually needed at runtime, with node_modules already pruned to
  // just the production dependencies that survived tree-shaking.
  //
  // Without this, a Docker image would need to ship the ENTIRE node_modules
  // folder (hundreds of MB, including devDependencies) to run `next start`.
  // The standalone build is what the official Next.js Docker example uses,
  // and it is what frontend/Dockerfile's runtime stage expects to find.
  output: "standalone",
};

export default nextConfig;
