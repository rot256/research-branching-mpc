
```
p = 0xffffffffffffffc5
```


```
make -j 8 tldr
make -j 8 semi-party.x
./compile.py --prime=65537 bmpc
../mpc/bmpc ./semi-party.x -N 2 -I -p 1 bmpc
../mpc/bmpc ./semi-party.x -N 2 -I -p 0 bmpc
```
