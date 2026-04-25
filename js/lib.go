package js

const requireRuntime = `
const __cache = {};

function require(modulePath) {
    return __require(modulePath, __rootDir);
}

function __require(modulePath, currentDir) {

    const resolved = __resolve(modulePath, currentDir);

    if (__cache[resolved]) {
        return __cache[resolved];
    }

    const code = __load(resolved);

    const module = { exports: {} };

    const dirname = resolved.substring(0, resolved.lastIndexOf("/"));

    const wrapped = "(function(module, exports, require, __filename, __dirname){ " + code + " })";

    const fn = eval(wrapped);

    function localRequire(p) {
        return __require(p, dirname);
    }

    fn(module, module.exports, localRequire, resolved, dirname);

    __cache[resolved] = module.exports;

    return module.exports;
}
`
