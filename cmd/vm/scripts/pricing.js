const math = require("./math");

exports.calcular = (precio, impuesto) => {
  return math.sumar(precio, impuesto);
};
