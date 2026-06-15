metadata({
  author: "César Galvis León",
  version: "1.0.0",
  description: "Pruebas de vm",
  license: "MIT",
  path: "https://github.com/cgalvisleon/jrex",
});

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

const users = db.GetModel("apps", "users");
users.BeforeInsert(function (tx, old, next) {
  const id = ULID();
  const now = timeNow();
  next.Set("created_at", now);
  next.Set("updated_at", now);
  next.Set("id", id);
  return null;
});

const item = users
  .Insert({
    name: "César Galvis León",
    email: "cgalvisleon@gmail.com",
    password: "123456",
  })
  .BeforeInsert(function (tx, old, next) {
    return null;
  })
  .Exec();

console.log(item.ToString());

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
