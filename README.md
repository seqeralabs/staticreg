```bash
docker run -d -p 5000:5000 --name registry registry
docker pull alpine
docker tag alpine localhost:5000/alpine
docker push localhost:5000/alpine
```
