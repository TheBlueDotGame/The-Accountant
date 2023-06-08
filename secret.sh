hash=$(echo -n ${date +%s} | shasum -a 256)
echo "hash: ${hash}"
