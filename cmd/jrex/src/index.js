author("César Galvis León");
description("Pruebas de vm");

const pricing = require("./modules/pricing");

// Test 1
result = pricing.calcular(100, 20);
console.log(result);

// Test 2
result = pricing.calcular(130, 19);
console.log(result);

// Test 3
result = pricing.multiplicar(120, 7);
console.log(result);
