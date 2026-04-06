const math = require("./math");

exports.calcular = (precio, impuesto) => {
  return math.sumar(precio, impuesto);
};

exports.multiplicar = (a, b) => {
  return math.multiplicar(a, b);
};
