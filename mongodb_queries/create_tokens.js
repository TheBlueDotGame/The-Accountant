// run with: mongo < create_tokens.js or VScode extension: MongoDB for VS Code
use('accountant');

db.getCollection('tokens').insertOne({"token": "80fda91a43989fa81347aa011e0f1e0fdde4eaabb408bf426166a62c80456c30","valid": true,"expiration_date": 1957894000000000000});
 