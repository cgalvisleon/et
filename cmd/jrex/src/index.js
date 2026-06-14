author("César Galvis León");
description("Pruebas de vm");

const pricing = require("./modules/pricing");

// Test 1
result = pricing.calcular(100, 15);
console.log(result);

// Test 2
result = pricing.calcular(130, 19);
console.log(result);

// Test 3
result = pricing.multiplicar(120, 7);
console.log(result);

ctx.set("name", "César Galvis León");
ctx.set("age", 30);
ctx.set("isMarried", true);
ctx.set("birthDate", new Date("1996-01-01"));
ctx.set("address", {
  street: "123 Main St",
  city: "Anytown",
  state: "CA",
  zip: "12345",
});
