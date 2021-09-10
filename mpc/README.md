
```
p = 0xffffffffffffffc5
```


```
make -j 8 tldr
make -j8 semi-party.x
./compile.py --prime=18446744073709551557 testing
./semi-party.x -N 2 -I -p 0 testing
./semi-party.x -N 2 -I -p 1 testing
```
