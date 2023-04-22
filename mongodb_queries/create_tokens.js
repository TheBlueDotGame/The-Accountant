// run with: mongo < create_tokens.js or VScode extension: MongoDB for VS Code
use('accountant');

db.getCollection('tokens')
  .insertOne(
    {
      "token": "wpg6d0grqJjyRicC8oI0/w6IGivm5ypFNTO/wwPGW9A=",
      "valid": true,
      "expiration_date": 1957894000000000000
    });
 