const { getDefaultConfig, mergeConfig } = require('@react-native/metro-config');
const path = require('path');
const fs = require('fs');

/**
 * Metro configuration
 * https://reactnative.dev/docs/metro
 *
 * @type {import('@react-native/metro-config').MetroConfig}
 */

const sharedPackageDir = path.resolve(__dirname, '../packages/shared');
const sharedSrcDir = path.join(sharedPackageDir, 'src');
const sharedIndex = path.join(sharedSrcDir, 'index.ts');

// Resolve `@chorus/shared` (a scoped package that lives outside node_modules in
// the `packages/` monorepo folder) to its TypeScript sources, and forward
// everything else to Metro's default resolver.
//
// Scoped packages cannot be mapped with `resolver.extraNodeModules` alone
// because Metro's resolver only keys extraNodeModules by the first path
// segment (`@chorus`), so `@chorus/shared` would be looked up as
// `packages/shared/@chorus/shared` and fail.
function resolveRequest(context, moduleName, platform) {
  if (moduleName === '@chorus/shared' || moduleName.startsWith('@chorus/shared/')) {
    let filePath;
    if (moduleName === '@chorus/shared') {
      filePath = sharedIndex;
    } else {
      const sub = moduleName.slice('@chorus/shared/'.length);
      const base = path.join(sharedSrcDir, sub);
      // Try common source extensions (and an index file) for deep imports.
      const candidates = [
        base,
        `${base}.ts`,
        `${base}.tsx`,
        `${base}.js`,
        `${base}.jsx`,
        path.join(base, 'index.ts'),
        path.join(base, 'index.js'),
      ];
      filePath = candidates.find(candidate => fs.existsSync(candidate));
      if (!filePath) {
        throw new Error(
          `Unable to resolve ${moduleName}. Could not find a module under ${sharedSrcDir}.`
        );
      }
    }
    return { filePath, type: 'sourceFile' };
  }
  // Forward the platform so the default resolver can still pick
  // platform-specific files (e.g. Foo.android.js) for other modules.
  return context.resolveRequest(context, moduleName, platform);
}

const config = {
  // The shared package lives outside the mobile project root, so it must be
  // registered as a watch folder for Metro to hash/transform its files.
  watchFolders: [sharedPackageDir],
  resolver: {
    resolveRequest,
    // Ensure modules imported from inside the shared package (e.g. axios,
    // @babel/runtime) resolve against the mobile project's own
    // node_modules, since the shared package has no node_modules of its own
    // and sits outside the mobile project root.
    nodeModulesPaths: [path.resolve(__dirname, 'node_modules')],
    // Block auto-generated KSV/regenerated paths from native module builds
    // that don't exist at dev time but can trigger Metro watch failures
    // (ENOENT on @react-native-async-storage/async-storage/android/build/*).
    blockList: [/\/build\/generated\/ksp\/.*/],
  },
};

module.exports = mergeConfig(getDefaultConfig(__dirname), config);