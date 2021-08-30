set_random_seed(0x13371337)

DIM = 12

def gen_prime(bits, unity):
    msb = 2^bits
    qou = floor(msb / unity)
    while 1:
        q = next_prime(randrange(1, qou))
        p = unity * q + 1
        if is_prime(p) and int(p).bit_length() == bits:
            return p

def gen_field(bits):
    p = gen_prime(bits, 2^DIM)
    F = GF(p)
    w = F.multiplicative_generator()
    c = (p - 1) / 2^DIM
    return (F, p, w^c)

(F1, q1, w1) = gen_field(46)
(F2, q2, w2) = gen_field(60)

