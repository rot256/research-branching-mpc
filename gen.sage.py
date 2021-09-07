

# This file was *autogenerated* from the file gen.sage
from sage.all_cmdline import *   # import sage library

_sage_const_0x13371337 = Integer(0x13371337); _sage_const_12 = Integer(12); _sage_const_2 = Integer(2); _sage_const_1 = Integer(1); _sage_const_46 = Integer(46); _sage_const_60 = Integer(60)
set_random_seed(_sage_const_0x13371337 )

DIM = _sage_const_12 

def gen_prime(bits, unity):
    msb = _sage_const_2 **bits
    qou = floor(msb / unity)
    while _sage_const_1 :
        q = next_prime(randrange(_sage_const_1 , qou))
        p = unity * q + _sage_const_1 
        if is_prime(p) and int(p).bit_length() == bits:
            return p

def gen_field(bits):
    p = gen_prime(bits, _sage_const_2 **DIM)
    F = GF(p)
    w = F.multiplicative_generator()
    c = (p - _sage_const_1 ) / _sage_const_2 **DIM
    return (F, p, w**c)

(F1, q1, w1) = gen_field(_sage_const_46 )
(F2, q2, w2) = gen_field(_sage_const_60 )

print(q1, q2)


