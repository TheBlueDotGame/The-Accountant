secret=$(pwgen -N 1 -s 96)
echo secret: ${secret}
hash=$(echo -n ${secret} | shasum -a 256)
echo "hash: ${hash}"