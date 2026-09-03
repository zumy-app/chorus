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

function resolveRequest(context, moduleName, platform) {
  if (moduleName === '@chorus/shared' || moduleName.startsWith('@chorus/shared/')) {
    let filePath;
    if (moduleName === '@chorus/shared') {
      filePath = sharedIndex;
    } else {
      const sub = moduleName.slice('@chorus/shared/'.length);
      const base = path.join(sharedSrcDir, sub);
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
  return context.resolveRequest(context, moduleName, platform);
}

const config = {
  watchFolders: [sharedPackageDir],
  resolver: {
    resolveRequest,
    nodeModulesPaths: [path.resolve(__dirname, 'node_modules')],
    blockList: [/\/build\/generated\/ksp\/.*/],
  },
};

module.exports = mergeConfig(getDefaultConfig(__dirname), config);
