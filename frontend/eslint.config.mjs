import { defineConfig, globalIgnores } from "eslint/config";
import nextPlugin from "@next/eslint-plugin-next";
import tsParser from "@typescript-eslint/parser";
import tsPlugin from "@typescript-eslint/eslint-plugin";
import reactPlugin from "eslint-plugin-react";
import reactHooks from "eslint-plugin-react-hooks";

/**
 * Flat ESLint config.
 *
 * The scaffold imported `eslint-config-next/core-web-vitals`, but the installed
 * eslint-config-next@15.5.21 ships *legacy eslintrc* configs (they rely on
 * `plugin:` strings and @rushstack/eslint-patch), which ESLint 9 flat config
 * cannot consume. Rather than bridge them with FlatCompat — which still loads
 * the incompatible patch — we compose the underlying plugins directly.
 *
 * Every plugin below was already installed as a transitive dependency of
 * eslint-config-next; no new packages were added.
 */
const eslintConfig = defineConfig([
  globalIgnores([".next/**", "out/**", "build/**", "next-env.d.ts"]),

  {
    files: ["**/*.{js,mjs,cjs,ts,tsx}"],
    languageOptions: {
      parser: tsParser,
      parserOptions: {
        ecmaFeatures: { jsx: true },
        sourceType: "module",
      },
    },
    plugins: {
      "@next/next": nextPlugin,
      "@typescript-eslint": tsPlugin,
      react: reactPlugin,
      "react-hooks": reactHooks,
    },
    settings: {
      react: { version: "detect" },
    },
    rules: {
      ...nextPlugin.configs.recommended.rules,
      ...nextPlugin.configs["core-web-vitals"].rules,
      ...tsPlugin.configs.recommended.rules,
      ...reactHooks.configs.recommended.rules,
      "react/jsx-key": "error",

      // CLAUDE.md §7: `any` requires a documented reason, so it must surface.
      "@typescript-eslint/no-explicit-any": "error",
      "@typescript-eslint/no-unused-vars": [
        "error",
        { argsIgnorePattern: "^_", varsIgnorePattern: "^_" },
      ],
    },
  },
]);

export default eslintConfig;
