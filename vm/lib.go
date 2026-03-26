package vm

const requireRuntime = `
const __moduleCache = {};

function require(path) {
    if (!path.endsWith(".js")) {
        path += ".js";
    }

    if (__moduleCache[path]) {
        return __moduleCache[path];
    }

    const code = __loadModule(path);

    const module = { exports: {} };

    const dirname = path.includes("/") 
        ? path.substring(0, path.lastIndexOf("/")) 
        : "";

    const wrapped = "(function(module, exports, require, __filename, __dirname) {" 
        + code + "\n})";

    const fn = eval(wrapped);

    function localRequire(p) {
        if (p.startsWith("./")) {
            return require(dirname + "/" + p.slice(2));
        }
        return require(p);
    }

    fn(module, module.exports, localRequire, path, dirname);

    __moduleCache[path] = module.exports;

    return module.exports;
}
`
