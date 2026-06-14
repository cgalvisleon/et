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

ctx.set({
  name: "César Galvis León",
  age: 30,
  isMarried: true,
  birthDate: new Date("1996-01-01"),
  address: {
    street: "Cra 12A # 3 - 23",
    city: "San Gil",
    state: "Santander",
    zip: "675001",
    country: "Colombia",
    countryCode: "CO",
    type: "home",
  },
});
