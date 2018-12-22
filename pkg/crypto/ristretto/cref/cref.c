#include "cref.h"

// -- scalar.c --


const group_scalar group_scalar_zero  = {{0}};
const group_scalar group_scalar_one   = {{1}};

static const crypto_uint32 m[32] = {0xED, 0xD3, 0xF5, 0x5C, 0x1A, 0x63, 0x12, 0x58, 0xD6, 0x9C, 0xF7, 0xA2, 0xDE, 0xF9, 0xDE, 0x14, 
                                    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x10};

static const crypto_uint32 mu[33] = {0x1B, 0x13, 0x2C, 0x0A, 0xA3, 0xE5, 0x9C, 0xED, 0xA7, 0x29, 0x63, 0x08, 0x5D, 0x21, 0x06, 0x21, 
                                     0xEB, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0x0F};

static crypto_uint32 lt(crypto_uint32 a,crypto_uint32 b) /* 16-bit inputs */
{
  unsigned int x = a;
  x -= (unsigned int) b; /* 0..65535: no; 4294901761..4294967295: yes */
  x >>= 31; /* 0: no; 1: yes */
  return x;
}

/* Reduce coefficients of r before calling reduce_add_sub */
static void reduce_add_sub(group_scalar *r)
{
  crypto_uint32 pb = 0;
  crypto_uint32 b;
  crypto_uint32 mask;
  int i;
  unsigned char t[32];

  for(i=0;i<32;i++) 
  {
    pb += m[i];
    b = lt(r->v[i],pb);
    t[i] = r->v[i]-pb+(b<<8);
    pb = b;
  }
  mask = b - 1;
  for(i=0;i<32;i++) 
    r->v[i] ^= mask & (r->v[i] ^ t[i]);
}

/* Reduce coefficients of x before calling barrett_reduce */
static void barrett_reduce(group_scalar *r, const crypto_uint32 x[64])
{
  /* See HAC, Alg. 14.42 */
  int i,j;
  crypto_uint32 q2[66];
  crypto_uint32 *q3 = q2 + 33;
  crypto_uint32 r1[33];
  crypto_uint32 r2[33];
  crypto_uint32 carry;
  crypto_uint32 pb = 0;
  crypto_uint32 b;

  for (i = 0;i < 66;++i) q2[i] = 0;
  for (i = 0;i < 33;++i) r2[i] = 0;

  for(i=0;i<33;i++)
    for(j=0;j<33;j++)
      if(i+j >= 31) q2[i+j] += mu[i]*x[j+31];
  carry = q2[31] >> 8;
  q2[32] += carry;
  carry = q2[32] >> 8;
  q2[33] += carry;

  for(i=0;i<33;i++)r1[i] = x[i];
  for(i=0;i<32;i++)
    for(j=0;j<33;j++)
      if(i+j < 33) r2[i+j] += m[i]*q3[j];

  for(i=0;i<32;i++)
  {
    carry = r2[i] >> 8;
    r2[i+1] += carry;
    r2[i] &= 0xff;
  }

  for(i=0;i<32;i++) 
  {
    pb += r2[i];
    b = lt(r1[i],pb);
    r->v[i] = r1[i]-pb+(b<<8);
    pb = b;
  }

  /* XXX: Can it really happen that r<0?, See HAC, Alg 14.42, Step 3 
   * If so: Handle  it here!
   */

  reduce_add_sub(r);
  reduce_add_sub(r);
}

int  group_scalar_unpack(group_scalar *r, const unsigned char x[GROUP_SCALAR_PACKEDBYTES])
{
  int i;
  for(i=0;i<32;i++)
    r->v[i] = x[i];
  r->v[31] &= 0x1f;
  reduce_add_sub(r);
  return 0;
}

void group_scalar_pack(unsigned char r[GROUP_SCALAR_PACKEDBYTES], const group_scalar *x)
{
  int i;
  for(i=0;i<32;i++)
    r[i] = x->v[i];
}

void group_scalar_setzero(group_scalar *r)
{
  int i;
  for(i=0;i<32;i++)
    r->v[i] = 0;
}

void group_scalar_setone(group_scalar *r)
{
  int i;
  r->v[0] = 1;
  for(i=1;i<32;i++)
    r->v[i] = 0;
}

/* Removed to avoid dependency on platform specific randombytes
void group_scalar_setrandom(group_scalar *r)
{
  unsigned char t[64];
  crypto_uint32 s[64];
  int i;
  randombytes(t,64);
  for(i=0;i<64;i++)
    s[i] = t[i];
  barrett_reduce(r,s);
}
*/

void group_scalar_add(group_scalar *r, const group_scalar *x, const group_scalar *y)
{
  int i, carry;
  for(i=0;i<32;i++) r->v[i] = x->v[i] + y->v[i];
  for(i=0;i<31;i++)
  {
    carry = r->v[i] >> 8;
    r->v[i+1] += carry;
    r->v[i] &= 0xff;
  }
  reduce_add_sub(r);
}

void group_scalar_sub(group_scalar *r, const group_scalar *x, const group_scalar *y)
{
  crypto_uint32 b = 0;
  crypto_uint32 t;
  int i;
  group_scalar d;

  for(i=0;i<32;i++)
  {
    t = m[i] - y->v[i] - b;
    d.v[i] = t & 255;
    b = (t >> 8) & 1;
  }
  group_scalar_add(r,x,&d);
}

void group_scalar_negate(group_scalar *r, const group_scalar *x)
{
  group_scalar t;
  group_scalar_setzero(&t);
  group_scalar_sub(r,&t,x);
}

void group_scalar_mul(group_scalar *r, const group_scalar *x, const group_scalar *y)
{
  int i,j,carry;
  crypto_uint32 t[64];
  for(i=0;i<64;i++)t[i] = 0;

  for(i=0;i<32;i++)
    for(j=0;j<32;j++)
      t[i+j] += x->v[i] * y->v[j];

  /* Reduce coefficients */
  for(i=0;i<63;i++)
  {
    carry = t[i] >> 8;
    t[i+1] += carry;
    t[i] &= 0xff;
  }

  barrett_reduce(r, t);
}

void group_scalar_square(group_scalar *r, const group_scalar *x)
{
  group_scalar_mul(r,x,x);
}

void group_scalar_invert(group_scalar *r, const group_scalar *x)
{
  group_scalar t0, t1, t2, t3, t4, t5;
  int i;

  group_scalar_square(&t1, x);
  group_scalar_mul(&t2, x, &t1);
  group_scalar_mul(&t0, &t1, &t2);
  group_scalar_square(&t1, &t0);
  group_scalar_square(&t3, &t1);
  group_scalar_mul(&t1, &t2, &t3);
  group_scalar_square(&t2, &t1);
  group_scalar_mul(&t3, &t0, &t2);
  group_scalar_square(&t0, &t3);
  group_scalar_mul(&t2, &t1, &t0);
  group_scalar_square(&t0, &t2);
  group_scalar_mul(&t1, &t2, &t0);
  group_scalar_square(&t0, &t1);
  group_scalar_mul(&t1, &t3, &t0);
  group_scalar_square(&t0, &t1);
  group_scalar_square(&t3, &t0);
  group_scalar_mul(&t0, &t1, &t3);
  group_scalar_mul(&t3, &t2, &t0);
  group_scalar_square(&t0, &t3);
  group_scalar_mul(&t2, &t1, &t0);
  group_scalar_square(&t0, &t2);
  group_scalar_mul(&t1, &t3, &t0);
  group_scalar_square(&t0, &t1);
  group_scalar_mul(&t3, &t1, &t0);
  group_scalar_mul(&t0, &t2, &t3);
  group_scalar_mul(&t2, &t1, &t0);
  group_scalar_square(&t1, &t2);
  group_scalar_square(&t3, &t1);
  group_scalar_square(&t4, &t3);
  group_scalar_mul(&t3, &t1, &t4);
  group_scalar_mul(&t1, &t0, &t3);
  group_scalar_mul(&t0, &t2, &t1);
  group_scalar_mul(&t2, &t1, &t0);
  group_scalar_square(&t1, &t2);
  group_scalar_square(&t3, &t1);
  group_scalar_mul(&t1, &t0, &t3);
  group_scalar_square(&t0, &t1);
  group_scalar_square(&t3, &t0);
  group_scalar_mul(&t0, &t1, &t3);
  group_scalar_mul(&t3, &t2, &t0);
  group_scalar_square(&t0, &t3);
  group_scalar_mul(&t2, &t1, &t0);
  group_scalar_square(&t0, &t2);
  group_scalar_square(&t1, &t0);
  group_scalar_mul(&t0, &t2, &t1);
  group_scalar_mul(&t1, &t3, &t0);
  group_scalar_square(&t0, &t1);
  group_scalar_square(&t3, &t0);
  group_scalar_square(&t0, &t3);
  group_scalar_square(&t3, &t0);
  group_scalar_square(&t0, &t3);
  group_scalar_square(&t3, &t0);
  group_scalar_mul(&t0, &t1, &t3);
  group_scalar_mul(&t3, &t2, &t0);
  group_scalar_square(&t0, &t3);
  group_scalar_mul(&t2, &t1, &t0);
  group_scalar_square(&t0, &t2);
  group_scalar_mul(&t1, &t2, &t0);
  group_scalar_square(&t0, &t1);
  group_scalar_mul(&t4, &t2, &t0);
  group_scalar_square(&t0, &t4);
  group_scalar_square(&t4, &t0);
  group_scalar_mul(&t0, &t1, &t4);
  group_scalar_mul(&t1, &t3, &t0);
  group_scalar_square(&t0, &t1);
  group_scalar_mul(&t3, &t1, &t0);
  group_scalar_square(&t0, &t3);
  group_scalar_square(&t4, &t0);
  group_scalar_mul(&t0, &t3, &t4);
  group_scalar_mul(&t3, &t2, &t0);
  group_scalar_square(&t0, &t3);
  group_scalar_square(&t2, &t0);
  group_scalar_square(&t0, &t2);
  group_scalar_mul(&t2, &t1, &t0);
  group_scalar_square(&t0, &t2);
  group_scalar_mul(&t1, &t3, &t0);
  group_scalar_mul(&t0, &t2, &t1);
  group_scalar_mul(&t2, &t1, &t0);
  group_scalar_square(&t1, &t2);
  group_scalar_square(&t3, &t1);
  group_scalar_mul(&t1, &t0, &t3);
  group_scalar_square(&t0, &t1);
  group_scalar_mul(&t3, &t2, &t0);
  group_scalar_mul(&t0, &t1, &t3);
  group_scalar_square(&t1, &t0);
  group_scalar_square(&t2, &t1);
  group_scalar_mul(&t1, &t0, &t2);
  group_scalar_mul(&t2, &t3, &t1);
  group_scalar_mul(&t1, &t0, &t2);
  group_scalar_mul(&t0, &t2, &t1);
  group_scalar_square(&t2, &t0);
  group_scalar_mul(&t3, &t0, &t2);
  group_scalar_square(&t2, &t3);
  group_scalar_mul(&t3, &t1, &t2);
  group_scalar_mul(&t1, &t0, &t3);
  group_scalar_square(&t0, &t1);
  group_scalar_mul(&t2, &t1, &t0);
  group_scalar_square(&t0, &t2);
  group_scalar_mul(&t4, &t2, &t0);
  group_scalar_square(&t0, &t4);
  group_scalar_square(&t4, &t0);
  group_scalar_square(&t5, &t4);
  group_scalar_square(&t4, &t5);
  group_scalar_square(&t5, &t4);
  group_scalar_square(&t4, &t5);
  group_scalar_mul(&t5, &t0, &t4);
  group_scalar_mul(&t0, &t2, &t5);
  group_scalar_mul(&t2, &t3, &t0);
  group_scalar_mul(&t0, &t1, &t2);
  group_scalar_square(&t1, &t0);
  group_scalar_mul(&t3, &t0, &t1);
  group_scalar_square(&t1, &t3);
  group_scalar_mul(&t4, &t0, &t1);
  group_scalar_square(&t1, &t4);
  group_scalar_square(&t4, &t1);
  group_scalar_square(&t1, &t4);
  group_scalar_mul(&t4, &t3, &t1);
  group_scalar_mul(&t1, &t2, &t4);
  group_scalar_square(&t2, &t1);
  group_scalar_square(&t3, &t2);
  group_scalar_square(&t4, &t3);
  group_scalar_mul(&t3, &t2, &t4);
  group_scalar_mul(&t2, &t1, &t3);
  group_scalar_mul(&t3, &t0, &t2);
  group_scalar_square(&t0, &t3);
  group_scalar_square(&t2, &t0);
  group_scalar_square(&t0, &t2);
  group_scalar_mul(&t2, &t1, &t0);
  group_scalar_mul(&t0, &t3, &t2);
  group_scalar_square(&t1, &t0);
  group_scalar_square(&t3, &t1);
  group_scalar_mul(&t4, &t1, &t3);
  group_scalar_square(&t3, &t4);
  group_scalar_square(&t4, &t3);
  group_scalar_mul(&t3, &t1, &t4);
  group_scalar_mul(&t1, &t2, &t3);
  group_scalar_square(&t2, &t1);
  group_scalar_square(&t3, &t2);
  group_scalar_mul(&t2, &t0, &t3);
  group_scalar_square(&t0, &t2);
  group_scalar_mul(&t3, &t1, &t0);
  group_scalar_square(&t0, &t3);
  group_scalar_mul(&t1, &t2, &t0);
  group_scalar_mul(&t0, &t3, &t1);
  group_scalar_square(&t2, &t0);
  group_scalar_square(&t3, &t2);
  group_scalar_square(&t2, &t3);
  group_scalar_square(&t3, &t2);
  group_scalar_mul(&t2, &t1, &t3);
  group_scalar_mul(&t1, &t0, &t2);
  group_scalar_square(&t0, &t1);
  group_scalar_square(&t3, &t0);
  group_scalar_square(&t4, &t3);
  group_scalar_mul(&t3, &t0, &t4);
  group_scalar_mul(&t0, &t1, &t3);
  group_scalar_mul(&t3, &t2, &t0);
  group_scalar_square(&t0, &t3);
  group_scalar_square(&t2, &t0);
  group_scalar_mul(&t0, &t1, &t2);
  group_scalar_square(&t1, &t0);
  group_scalar_mul(&t2, &t3, &t1);
  group_scalar_mul(&t1, &t0, &t2);
  group_scalar_square(&t0, &t1);
  group_scalar_mul(&t3, &t2, &t0);
  group_scalar_square(&t0, &t3);
  group_scalar_square(&t2, &t0);
  group_scalar_mul(&t0, &t1, &t2);
  group_scalar_mul(&t1, &t3, &t0);
  group_scalar_square(&t2, &t1);
  group_scalar_mul(&t3, &t0, &t2);
  group_scalar_mul(&t0, &t1, &t3);
  group_scalar_square(&t1, &t0);
  group_scalar_square(&t2, &t1);
  group_scalar_square(&t4, &t2);
  group_scalar_mul(&t2, &t1, &t4);
  group_scalar_square(&t4, &t2);
  group_scalar_square(&t2, &t4);
  group_scalar_square(&t4, &t2);
  group_scalar_mul(&t2, &t1, &t4);
  group_scalar_mul(&t1, &t3, &t2);
  group_scalar_square(&t2, &t1);
  group_scalar_square(&t3, &t2);
  group_scalar_mul(&t2, &t1, &t3);
  group_scalar_square(&t3, &t2);
  group_scalar_square(&t2, &t3);
  group_scalar_mul(&t3, &t1, &t2);
  group_scalar_mul(&t2, &t0, &t3);
  group_scalar_square(&t0, &t2);
  group_scalar_mul(&t3, &t2, &t0);
  group_scalar_square(&t0, &t3);
  group_scalar_square(&t4, &t0);
  group_scalar_mul(&t0, &t3, &t4);
  group_scalar_mul(&t3, &t1, &t0);
  group_scalar_square(&t0, &t3);
  group_scalar_mul(&t1, &t3, &t0);
  group_scalar_mul(&t0, &t2, &t1);
  for(i = 0; i < 126; i++)
    group_scalar_square(&t0, &t0);
  group_scalar_mul(r, &t3, &t0);
}

int  group_scalar_isone(const group_scalar *x)
{
  unsigned long long r;
  int i;
  r = 1-x->v[0];
  for(i=1;i<32;i++)
    r |= x->v[i];
  return 1-((-r)>>63);
}

int  group_scalar_iszero(const group_scalar *x)
{
  unsigned long long r=0;
  int i;
  for(i=0;i<32;i++)
    r |= x->v[i];
  return 1-((-r)>>63);
}

int  group_scalar_equals(const group_scalar *x,  const group_scalar *y)
{
  unsigned long long r=0;
  int i;
  for(i=0;i<32;i++)
    r |= (x->v[i] ^ y->v[i]);
  return 1-((-r)>>63);
}


// Additional functions, not required by API
int scalar_tstbit(const group_scalar *x, const unsigned int pos)
{
  return (x->v[pos >> 3] & (1ULL << (pos & 0x7))) >> (pos & 0x7);
}

int  scalar_bitlen(const group_scalar *x)
{
  int i;
  unsigned long long mask;
  int ctr = 256;
  int found = 0;
  int t;
  for(i=31;i>=0;i--)
  {
    for(mask = (1 << 7);mask>0;mask>>=1)
    {
      found = found || (mask & x->v[i]);
      t = ctr - 1;
      ctr = (found * ctr)^((1-found)*t);
    }
  }
  return ctr;
}

void scalar_window3(signed char r[85], const group_scalar *s)
{
  char carry;
  int i;
  for(i=0;i<10;i++)
  {
    r[8*i+0]  =  s->v[3*i+0]       & 7;
    r[8*i+1]  = (s->v[3*i+0] >> 3) & 7;
    r[8*i+2]  = (s->v[3*i+0] >> 6) & 7;
    r[8*i+2] ^= (s->v[3*i+1] << 2) & 7;
    r[8*i+3]  = (s->v[3*i+1] >> 1) & 7;
    r[8*i+4]  = (s->v[3*i+1] >> 4) & 7;
    r[8*i+5]  = (s->v[3*i+1] >> 7) & 7;
    r[8*i+5] ^= (s->v[3*i+2] << 1) & 7;
    r[8*i+6]  = (s->v[3*i+2] >> 2) & 7;
    r[8*i+7]  = (s->v[3*i+2] >> 5) & 7;
  }
  r[8*i+0]  =  s->v[3*i+0]       & 7;
  r[8*i+1]  = (s->v[3*i+0] >> 3) & 7;
  r[8*i+2]  = (s->v[3*i+0] >> 6) & 7;
  r[8*i+2] ^= (s->v[3*i+1] << 2) & 7;
  r[8*i+3]  = (s->v[3*i+1] >> 1) & 7;
  r[8*i+4]  = (s->v[3*i+1] >> 4) & 7;

  /* Making it signed */
  carry = 0;
  for(i=0;i<84;i++)
  {
    r[i] += carry;
    r[i+1] += r[i] >> 3;
    r[i] &= 7;
    carry = r[i] >> 2;
    r[i] -= carry<<3;
  }
  r[84] += carry;
}

void scalar_window5(signed char r[51], const group_scalar *s) 
{
  char carry;
  int i;
  for(i=0;i<6;i++)
  {
    r[8*i+0]  =  s->v[5*i+0] & 31;
    r[8*i+1]  = (s->v[5*i+0] >> 5) & 31;
    r[8*i+1] ^= (s->v[5*i+1] << 3) & 31;
    r[8*i+2]  = (s->v[5*i+1] >> 2) & 31;
    r[8*i+3]  = (s->v[5*i+1] >> 7) & 31;
    r[8*i+3] ^= (s->v[5*i+2] << 1) & 31;
    r[8*i+4]  = (s->v[5*i+2] >> 4) & 31;
    r[8*i+4] ^= (s->v[5*i+3] << 4) & 31;
    r[8*i+5]  = (s->v[5*i+3] >> 1) & 31;
    r[8*i+6]  = (s->v[5*i+3] >> 6) & 31;
    r[8*i+6] ^= (s->v[5*i+4] << 2) & 31;
    r[8*i+7]  = (s->v[5*i+4] >> 3) & 31;
  }
  r[8*i+0]  =  s->v[5*i+0] & 31;
  r[8*i+1]  = (s->v[5*i+0] >> 5) & 31;
  r[8*i+1] ^= (s->v[5*i+1] << 3) & 31;
  r[8*i+2]  = (s->v[5*i+1] >> 2) & 31;


  /* Making it signed */
  carry = 0;
  for(i=0;i<50;i++)
  {
    r[i] += carry;
    r[i+1] += r[i] >> 5;
    r[i] &= 31; 
    carry = r[i] >> 4;
    r[i] -= carry << 5;
  }
  r[50] += carry;
}

void scalar_slide(signed char r[256], const group_scalar *s, int swindowsize)
{
  int i,j,k,b,m=(1<<(swindowsize-1))-1, soplen=256;

  for(i=0;i<32;i++) 
  {
    r[8*i+0] =  s->v[i] & 1;
    r[8*i+1] = (s->v[i] >> 1) & 1;
    r[8*i+2] = (s->v[i] >> 2) & 1;
    r[8*i+3] = (s->v[i] >> 3) & 1;
    r[8*i+4] = (s->v[i] >> 4) & 1;
    r[8*i+5] = (s->v[i] >> 5) & 1;
    r[8*i+6] = (s->v[i] >> 6) & 1;
    r[8*i+7] = (s->v[i] >> 7) & 1;
  }

  /* Making it sliding window */
  for (j = 0;j < soplen;++j) 
  {
    if (r[j]) {
      for (b = 1;b < soplen - j && b <= 6;++b) {
        if (r[j] + (r[j + b] << b) <= m) 
        {
          r[j] += r[j + b] << b; r[j + b] = 0;
        } 
        else if (r[j] - (r[j + b] << b) >= -m) 
        {
          r[j] -= r[j + b] << b;
          for (k = j + b;k < soplen;++k) 
          {
            if (!r[k]) {
              r[k] = 1;
              break;
            }
            r[k] = 0;
          }
        } 
        else if (r[j + b])
          break;
      }
    }
  }
}
/*
void scalar_print(const group_scalar *x)
{
  int i;
  for(i=0;i<31;i++)
    printf("%d*2^(%d*8) + ",x->v[i],i);
  printf("%d*2^(%d*8)\n",x->v[i],i);
}
*/
void scalar_from64bytes(group_scalar *r, const unsigned char h[64])
{
  int i;
  crypto_uint32 t[64];
  for(i=0;i<64;i++) t[i] = h[i];
  barrett_reduce(r, t); 
}



// -- fe25519.c --

const fe25519 fe25519_zero = {{0, 0, 0, 0, 0, 0, 0, 0, 0, 0}};
const fe25519 fe25519_one = {{1, 0, 0, 0, 0, 0, 0, 0, 0, 0}};
const fe25519 fe25519_two = {{2, 0, 0, 0, 0, 0, 0, 0, 0, 0}};
const fe25519 fe25519_sqrtm1 = {{-32595792, -7943725, 9377950, 3500415, 12389472, -272473, -25146209, -2005654, 326686, 11406482}};
const fe25519 fe25519_msqrtm1 = {{32595792, 7943725, -9377950, -3500415, -12389472, 272473, 25146209, 2005654, -326686, -11406482}};
const fe25519 fe25519_m1 = {{-1, 0, 0, 0, 0, 0, 0, 0, 0, 0}};


static crypto_uint32 fe25519_c_static_equal(crypto_uint32 a,crypto_uint32 b) /* 16-bit inputs */
{
  crypto_uint32 x = a ^ b; /* 0: yes; 1..65535: no */
  x -= 1; /* 4294967295: yes; 0..65534: no */
  x >>= 31; /* 1: yes; 0: no */
  return x;
}

static crypto_uint64 load_3(const unsigned char *in)
{
  crypto_uint64 result;
  result = (crypto_uint64) in[0];
  result |= ((crypto_uint64) in[1]) << 8;
  result |= ((crypto_uint64) in[2]) << 16;
  return result;
}

static crypto_uint64 load_4(const unsigned char *in)
{
  crypto_uint64 result;
  result = (crypto_uint64) in[0];
  result |= ((crypto_uint64) in[1]) << 8;
  result |= ((crypto_uint64) in[2]) << 16;
  result |= ((crypto_uint64) in[3]) << 24;
  return result;
}

/*
 * Ignores top bit of h.
 */
void fe25519_unpack(fe25519 *h,const unsigned char s[32])
{
  crypto_int64 h0 = load_4(s);
  crypto_int64 h1 = load_3(s + 4) << 6;
  crypto_int64 h2 = load_3(s + 7) << 5;
  crypto_int64 h3 = load_3(s + 10) << 3;
  crypto_int64 h4 = load_3(s + 13) << 2;
  crypto_int64 h5 = load_4(s + 16);
  crypto_int64 h6 = load_3(s + 20) << 7;
  crypto_int64 h7 = load_3(s + 23) << 5;
  crypto_int64 h8 = load_3(s + 26) << 4;
  crypto_int64 h9 = (load_3(s + 29) & 8388607) << 2;
  crypto_int64 carry0;
  crypto_int64 carry1;
  crypto_int64 carry2;
  crypto_int64 carry3;
  crypto_int64 carry4;
  crypto_int64 carry5;
  crypto_int64 carry6;
  crypto_int64 carry7;
  crypto_int64 carry8;
  crypto_int64 carry9;
  
  carry9 = (h9 + (crypto_int64) (1<<24)) >> 25; h0 += carry9 * 19; h9 -= carry9 << 25;
  carry1 = (h1 + (crypto_int64) (1<<24)) >> 25; h2 += carry1; h1 -= carry1 << 25;
  carry3 = (h3 + (crypto_int64) (1<<24)) >> 25; h4 += carry3; h3 -= carry3 << 25;
  carry5 = (h5 + (crypto_int64) (1<<24)) >> 25; h6 += carry5; h5 -= carry5 << 25;
  carry7 = (h7 + (crypto_int64) (1<<24)) >> 25; h8 += carry7; h7 -= carry7 << 25;
  
  carry0 = (h0 + (crypto_int64) (1<<25)) >> 26; h1 += carry0; h0 -= carry0 << 26;
  carry2 = (h2 + (crypto_int64) (1<<25)) >> 26; h3 += carry2; h2 -= carry2 << 26;
  carry4 = (h4 + (crypto_int64) (1<<25)) >> 26; h5 += carry4; h4 -= carry4 << 26;
  carry6 = (h6 + (crypto_int64) (1<<25)) >> 26; h7 += carry6; h6 -= carry6 << 26;
  carry8 = (h8 + (crypto_int64) (1<<25)) >> 26; h9 += carry8; h8 -= carry8 << 26;
  
  h->v[0] = h0;
  h->v[1] = h1;
  h->v[2] = h2;
  h->v[3] = h3;
  h->v[4] = h4;
  h->v[5] = h5;
  h->v[6] = h6;
  h->v[7] = h7;
  h->v[8] = h8;
  h->v[9] = h9;
}


/*
 * Preconditions:
 *  |h| bounded by 1.1*2^26,1.1*2^25,1.1*2^26,1.1*2^25,etc.
 * 
 * Write p=2^255-19; q=floor(h/p).
 * Basic claim: q = floor(2^(-255)(h + 19 2^(-25)h9 + 2^(-1))).
 * 
 * Proof:
 *  Have |h|<=p so |q|<=1 so |19^2 2^(-255) q|<1/4.
 *  Also have |h-2^230 h9|<2^231 so |19 2^(-255)(h-2^230 h9)|<1/4.
 * 
 *  Write y=2^(-1)-19^2 2^(-255)q-19 2^(-255)(h-2^230 h9).
 *  Then 0<y<1.
 * 
 *  Write r=h-pq.
 *  Have 0<=r<=p-1=2^255-20.
 *  Thus 0<=r+19(2^-255)r<r+19(2^-255)2^255<=2^255-1.
 * 
 *  Write x=r+19(2^-255)r+y.
 *  Then 0<x<2^255 so floor(2^(-255)x) = 0 so floor(q+2^(-255)x) = q.
 * 
 *  Have q+2^(-255)x = 2^(-255)(h + 19 2^(-25) h9 + 2^(-1))
 *  so floor(2^(-255)(h + 19 2^(-25) h9 + 2^(-1))) = q.
 */

void fe25519_pack(unsigned char s[32],const fe25519 *h)
{
  crypto_int32 h0 = h->v[0];
  crypto_int32 h1 = h->v[1];
  crypto_int32 h2 = h->v[2];
  crypto_int32 h3 = h->v[3];
  crypto_int32 h4 = h->v[4];
  crypto_int32 h5 = h->v[5];
  crypto_int32 h6 = h->v[6];
  crypto_int32 h7 = h->v[7];
  crypto_int32 h8 = h->v[8];
  crypto_int32 h9 = h->v[9];
  crypto_int32 q;
  crypto_int32 carry0;
  crypto_int32 carry1;
  crypto_int32 carry2;
  crypto_int32 carry3;
  crypto_int32 carry4;
  crypto_int32 carry5;
  crypto_int32 carry6;
  crypto_int32 carry7;
  crypto_int32 carry8;
  crypto_int32 carry9;
  
  q = (19 * h9 + (((crypto_int32) 1) << 24)) >> 25;
  q = (h0 + q) >> 26;
  q = (h1 + q) >> 25;
  q = (h2 + q) >> 26;
  q = (h3 + q) >> 25;
  q = (h4 + q) >> 26;
  q = (h5 + q) >> 25;
  q = (h6 + q) >> 26;
  q = (h7 + q) >> 25;
  q = (h8 + q) >> 26;
  q = (h9 + q) >> 25;
  
  /* Goal: Output h-(2^255-19)q, which is between 0 and 2^255-20. */
  h0 += 19 * q;
  /* Goal: Output h-2^255 q, which is between 0 and 2^255-20. */
  
  carry0 = h0 >> 26; h1 += carry0; h0 -= carry0 << 26;
  carry1 = h1 >> 25; h2 += carry1; h1 -= carry1 << 25;
  carry2 = h2 >> 26; h3 += carry2; h2 -= carry2 << 26;
  carry3 = h3 >> 25; h4 += carry3; h3 -= carry3 << 25;
  carry4 = h4 >> 26; h5 += carry4; h4 -= carry4 << 26;
  carry5 = h5 >> 25; h6 += carry5; h5 -= carry5 << 25;
  carry6 = h6 >> 26; h7 += carry6; h6 -= carry6 << 26;
  carry7 = h7 >> 25; h8 += carry7; h7 -= carry7 << 25;
  carry8 = h8 >> 26; h9 += carry8; h8 -= carry8 << 26;
  carry9 = h9 >> 25;               h9 -= carry9 << 25;
  /* h10 = carry9 */
  
  /*
   *  Goal: Output h0+...+2^255 h10-2^255 q, which is between 0 and 2^255-20.
   *  Have h0+...+2^230 h9 between 0 and 2^255-1;
   *  evidently 2^255 h10-2^255 q = 0.
   *  Goal: Output h0+...+2^230 h9.
   */
  
  s[0] = h0 >> 0;
  s[1] = h0 >> 8;
  s[2] = h0 >> 16;
  s[3] = (h0 >> 24) | (h1 << 2);
  s[4] = h1 >> 6;
  s[5] = h1 >> 14;
  s[6] = (h1 >> 22) | (h2 << 3);
  s[7] = h2 >> 5;
  s[8] = h2 >> 13;
  s[9] = (h2 >> 21) | (h3 << 5);
  s[10] = h3 >> 3;
  s[11] = h3 >> 11;
  s[12] = (h3 >> 19) | (h4 << 6);
  s[13] = h4 >> 2;
  s[14] = h4 >> 10;
  s[15] = h4 >> 18;
  s[16] = h5 >> 0;
  s[17] = h5 >> 8;
  s[18] = h5 >> 16;
  s[19] = (h5 >> 24) | (h6 << 1);
  s[20] = h6 >> 7;
  s[21] = h6 >> 15;
  s[22] = (h6 >> 23) | (h7 << 3);
  s[23] = h7 >> 5;
  s[24] = h7 >> 13;
  s[25] = (h7 >> 21) | (h8 << 4);
  s[26] = h8 >> 4;
  s[27] = h8 >> 12;
  s[28] = (h8 >> 20) | (h9 << 6);
  s[29] = h9 >> 2;
  s[30] = h9 >> 10;
  s[31] = h9 >> 18;
}

/*
 * return 1 if f == 0
 * return 0 if f != 0
 * 
 * Preconditions:
 *   |f| bounded by 1.1*2^26,1.1*2^25,1.1*2^26,1.1*2^25,etc.
 */
static const unsigned char zero[32];

int fe25519_iszero(const fe25519 *f)
{
  int i,r=0;
  unsigned char s[32];
  fe25519_pack(s,f);
  for(i=0;i<32;i++)
    r |= (1-fe25519_c_static_equal(zero[i],s[i]));
  return 1-r;
}

int fe25519_isone(const fe25519 *x) 
{
  return fe25519_iseq(x, &fe25519_one);  
}  

/*
 * return 1 if f is in {1,3,5,...,q-2}
 * return 0 if f is in {0,2,4,...,q-1}
 * 
 * Preconditions:
 *   |f| bounded by 1.1*2^26,1.1*2^25,1.1*2^26,1.1*2^25,etc.
 */

int fe25519_isnegative(const fe25519 *f)
{
  unsigned char s[32];
  fe25519_pack(s,f);
  return s[0] & 1;
}

int fe25519_iseq(const fe25519 *x, const fe25519 *y)
{
  fe25519 t;
  fe25519_sub(&t, x, y);
  return fe25519_iszero(&t);
}

int fe25519_iseq_vartime(const fe25519 *x, const fe25519 *y) {
  return fe25519_iseq(x, y);
}  

/*
 * Replace (f,g) with (g,g) if b == 1;
 * replace (f,g) with (f,g) if b == 0.
 * 
 * Preconditions: b in {0,1}.
 */

void fe25519_cmov(fe25519 *r, const fe25519 *x, unsigned char b)
{
  int i;
  crypto_uint32 mask = b;
  mask = -mask;
  for(i=0;i<10;i++) r->v[i] ^= mask & (x->v[i] ^ r->v[i]);
}


/*
 * h = 1
 */

void fe25519_setone(fe25519 *h)
{
  h->v[0] = 1;
  h->v[1] = 0;
  h->v[2] = 0;
  h->v[3] = 0;
  h->v[4] = 0;
  h->v[5] = 0;
  h->v[6] = 0;
  h->v[7] = 0;
  h->v[8] = 0;
  h->v[9] = 0;
}

/*
 * h = 0
 */

void fe25519_setzero(fe25519 *h)
{
  h->v[0] = 0;
  h->v[1] = 0;
  h->v[2] = 0;
  h->v[3] = 0;
  h->v[4] = 0;
  h->v[5] = 0;
  h->v[6] = 0;
  h->v[7] = 0;
  h->v[8] = 0;
  h->v[9] = 0;
}


/*
 * h = -f
 * 
 * Preconditions:
 *   |f| bounded by 1.1*2^25,1.1*2^24,1.1*2^25,1.1*2^24,etc.
 * 
 * Postconditions:
 *   |h| bounded by 1.1*2^25,1.1*2^24,1.1*2^25,1.1*2^24,etc.
 */

void fe25519_neg(fe25519 *h, const fe25519 *f)
{
  crypto_int32 f0 = f->v[0];
  crypto_int32 f1 = f->v[1];
  crypto_int32 f2 = f->v[2];
  crypto_int32 f3 = f->v[3];
  crypto_int32 f4 = f->v[4];
  crypto_int32 f5 = f->v[5];
  crypto_int32 f6 = f->v[6];
  crypto_int32 f7 = f->v[7];
  crypto_int32 f8 = f->v[8];
  crypto_int32 f9 = f->v[9];
  crypto_int32 h0 = -f0;
  crypto_int32 h1 = -f1;
  crypto_int32 h2 = -f2;
  crypto_int32 h3 = -f3;
  crypto_int32 h4 = -f4;
  crypto_int32 h5 = -f5;
  crypto_int32 h6 = -f6;
  crypto_int32 h7 = -f7;
  crypto_int32 h8 = -f8;
  crypto_int32 h9 = -f9;
  h->v[0] = h0;
  h->v[1] = h1;
  h->v[2] = h2;
  h->v[3] = h3;
  h->v[4] = h4;
  h->v[5] = h5;
  h->v[6] = h6;
  h->v[7] = h7;
  h->v[8] = h8;
  h->v[9] = h9;
}


unsigned char fe25519_getparity(const fe25519 *x) {
  return fe25519_isnegative(x);  
}  


/*
 * h = f + g
 * Can overlap h with f or g.
 * 
 * Preconditions:
 *   |f| bounded by 1.1*2^25,1.1*2^24,1.1*2^25,1.1*2^24,etc.
 *   |g| bounded by 1.1*2^25,1.1*2^24,1.1*2^25,1.1*2^24,etc.
 * 
 * Postconditions:
 *   |h| bounded by 1.1*2^26,1.1*2^25,1.1*2^26,1.1*2^25,etc.
 */

void fe25519_add(fe25519 *h,const fe25519 *f,const fe25519 *g)
{
  crypto_int32 f0 = f->v[0];
  crypto_int32 f1 = f->v[1];
  crypto_int32 f2 = f->v[2];
  crypto_int32 f3 = f->v[3];
  crypto_int32 f4 = f->v[4];
  crypto_int32 f5 = f->v[5];
  crypto_int32 f6 = f->v[6];
  crypto_int32 f7 = f->v[7];
  crypto_int32 f8 = f->v[8];
  crypto_int32 f9 = f->v[9];
  crypto_int32 g0 = g->v[0];
  crypto_int32 g1 = g->v[1];
  crypto_int32 g2 = g->v[2];
  crypto_int32 g3 = g->v[3];
  crypto_int32 g4 = g->v[4];
  crypto_int32 g5 = g->v[5];
  crypto_int32 g6 = g->v[6];
  crypto_int32 g7 = g->v[7];
  crypto_int32 g8 = g->v[8];
  crypto_int32 g9 = g->v[9];
  crypto_int32 h0 = f0 + g0;
  crypto_int32 h1 = f1 + g1;
  crypto_int32 h2 = f2 + g2;
  crypto_int32 h3 = f3 + g3;
  crypto_int32 h4 = f4 + g4;
  crypto_int32 h5 = f5 + g5;
  crypto_int32 h6 = f6 + g6;
  crypto_int32 h7 = f7 + g7;
  crypto_int32 h8 = f8 + g8;
  crypto_int32 h9 = f9 + g9;
  h->v[0] = h0;
  h->v[1] = h1;
  h->v[2] = h2;
  h->v[3] = h3;
  h->v[4] = h4;
  h->v[5] = h5;
  h->v[6] = h6;
  h->v[7] = h7;
  h->v[8] = h8;
  h->v[9] = h9;
}


void fe25519_double(fe25519 *r, const fe25519 *x) {
  fe25519_add(r, x, x);  
}

void fe25519_triple(fe25519 *r, const fe25519 *x) {
  fe25519_add(r, x, x);
  fe25519_add(r, r, x);
}

/*
 * h = f - g
 * Can overlap h with f or g.
 * 
 * Preconditions:
 *   |f| bounded by 1.1*2^25,1.1*2^24,1.1*2^25,1.1*2^24,etc.
 *   |g| bounded by 1.1*2^25,1.1*2^24,1.1*2^25,1.1*2^24,etc.
 * 
 * Postconditions:
 *   |h| bounded by 1.1*2^26,1.1*2^25,1.1*2^26,1.1*2^25,etc.
 */

void fe25519_sub(fe25519 *h,const fe25519 *f,const fe25519 *g)
{
  crypto_int32 f0 = f->v[0];
  crypto_int32 f1 = f->v[1];
  crypto_int32 f2 = f->v[2];
  crypto_int32 f3 = f->v[3];
  crypto_int32 f4 = f->v[4];
  crypto_int32 f5 = f->v[5];
  crypto_int32 f6 = f->v[6];
  crypto_int32 f7 = f->v[7];
  crypto_int32 f8 = f->v[8];
  crypto_int32 f9 = f->v[9];
  crypto_int32 g0 = g->v[0];
  crypto_int32 g1 = g->v[1];
  crypto_int32 g2 = g->v[2];
  crypto_int32 g3 = g->v[3];
  crypto_int32 g4 = g->v[4];
  crypto_int32 g5 = g->v[5];
  crypto_int32 g6 = g->v[6];
  crypto_int32 g7 = g->v[7];
  crypto_int32 g8 = g->v[8];
  crypto_int32 g9 = g->v[9];
  crypto_int32 h0 = f0 - g0;
  crypto_int32 h1 = f1 - g1;
  crypto_int32 h2 = f2 - g2;
  crypto_int32 h3 = f3 - g3;
  crypto_int32 h4 = f4 - g4;
  crypto_int32 h5 = f5 - g5;
  crypto_int32 h6 = f6 - g6;
  crypto_int32 h7 = f7 - g7;
  crypto_int32 h8 = f8 - g8;
  crypto_int32 h9 = f9 - g9;
  h->v[0] = h0;
  h->v[1] = h1;
  h->v[2] = h2;
  h->v[3] = h3;
  h->v[4] = h4;
  h->v[5] = h5;
  h->v[6] = h6;
  h->v[7] = h7;
  h->v[8] = h8;
  h->v[9] = h9;
}


/*
 * h = f * g
 * Can overlap h with f or g.
 * 
 * Preconditions:
 *   |f| bounded by 1.65*2^26,1.65*2^25,1.65*2^26,1.65*2^25,etc.
 *   |g| bounded by 1.65*2^26,1.65*2^25,1.65*2^26,1.65*2^25,etc.
 * 
 * Postconditions:
 *   |h| bounded by 1.01*2^25,1.01*2^24,1.01*2^25,1.01*2^24,etc.
 */

/*
 * Notes on implementation strategy:
 * 
 * Using schoolbook multiplication.
 * Karatsuba would save a little in some cost models.
 * 
 * Most multiplications by 2 and 19 are 32-bit precomputations;
 * cheaper than 64-bit postcomputations.
 * 
 * There is one remaining multiplication by 19 in the carry chain;
 * one *19 precomputation can be merged into this,
 * but the resulting data flow is considerably less clean.
 * 
 * There are 12 carries below.
 * 10 of them are 2-way parallelizable and vectorizable.
 * Can get away with 11 carries, but then data flow is much deeper.
 * 
 * With tighter constraints on inputs can squeeze carries into int32.
 */

void fe25519_mul(fe25519 *h,const fe25519 *f,const fe25519 *g)
{
  crypto_int32 f0 = f->v[0];
  crypto_int32 f1 = f->v[1];
  crypto_int32 f2 = f->v[2];
  crypto_int32 f3 = f->v[3];
  crypto_int32 f4 = f->v[4];
  crypto_int32 f5 = f->v[5];
  crypto_int32 f6 = f->v[6];
  crypto_int32 f7 = f->v[7];
  crypto_int32 f8 = f->v[8];
  crypto_int32 f9 = f->v[9];
  crypto_int32 g0 = g->v[0];
  crypto_int32 g1 = g->v[1];
  crypto_int32 g2 = g->v[2];
  crypto_int32 g3 = g->v[3];
  crypto_int32 g4 = g->v[4];
  crypto_int32 g5 = g->v[5];
  crypto_int32 g6 = g->v[6];
  crypto_int32 g7 = g->v[7];
  crypto_int32 g8 = g->v[8];
  crypto_int32 g9 = g->v[9];
  crypto_int32 g1_19 = 19 * g1; /* 1.959375*2^29 */
  crypto_int32 g2_19 = 19 * g2; /* 1.959375*2^30; still ok */
  crypto_int32 g3_19 = 19 * g3;
  crypto_int32 g4_19 = 19 * g4;
  crypto_int32 g5_19 = 19 * g5;
  crypto_int32 g6_19 = 19 * g6;
  crypto_int32 g7_19 = 19 * g7;
  crypto_int32 g8_19 = 19 * g8;
  crypto_int32 g9_19 = 19 * g9;
  crypto_int32 f1_2 = 2 * f1;
  crypto_int32 f3_2 = 2 * f3;
  crypto_int32 f5_2 = 2 * f5;
  crypto_int32 f7_2 = 2 * f7;
  crypto_int32 f9_2 = 2 * f9;
  crypto_int64 f0g0    = f0   * (crypto_int64) g0;
  crypto_int64 f0g1    = f0   * (crypto_int64) g1;
  crypto_int64 f0g2    = f0   * (crypto_int64) g2;
  crypto_int64 f0g3    = f0   * (crypto_int64) g3;
  crypto_int64 f0g4    = f0   * (crypto_int64) g4;
  crypto_int64 f0g5    = f0   * (crypto_int64) g5;
  crypto_int64 f0g6    = f0   * (crypto_int64) g6;
  crypto_int64 f0g7    = f0   * (crypto_int64) g7;
  crypto_int64 f0g8    = f0   * (crypto_int64) g8;
  crypto_int64 f0g9    = f0   * (crypto_int64) g9;
  crypto_int64 f1g0    = f1   * (crypto_int64) g0;
  crypto_int64 f1g1_2  = f1_2 * (crypto_int64) g1;
  crypto_int64 f1g2    = f1   * (crypto_int64) g2;
  crypto_int64 f1g3_2  = f1_2 * (crypto_int64) g3;
  crypto_int64 f1g4    = f1   * (crypto_int64) g4;
  crypto_int64 f1g5_2  = f1_2 * (crypto_int64) g5;
  crypto_int64 f1g6    = f1   * (crypto_int64) g6;
  crypto_int64 f1g7_2  = f1_2 * (crypto_int64) g7;
  crypto_int64 f1g8    = f1   * (crypto_int64) g8;
  crypto_int64 f1g9_38 = f1_2 * (crypto_int64) g9_19;
  crypto_int64 f2g0    = f2   * (crypto_int64) g0;
  crypto_int64 f2g1    = f2   * (crypto_int64) g1;
  crypto_int64 f2g2    = f2   * (crypto_int64) g2;
  crypto_int64 f2g3    = f2   * (crypto_int64) g3;
  crypto_int64 f2g4    = f2   * (crypto_int64) g4;
  crypto_int64 f2g5    = f2   * (crypto_int64) g5;
  crypto_int64 f2g6    = f2   * (crypto_int64) g6;
  crypto_int64 f2g7    = f2   * (crypto_int64) g7;
  crypto_int64 f2g8_19 = f2   * (crypto_int64) g8_19;
  crypto_int64 f2g9_19 = f2   * (crypto_int64) g9_19;
  crypto_int64 f3g0    = f3   * (crypto_int64) g0;
  crypto_int64 f3g1_2  = f3_2 * (crypto_int64) g1;
  crypto_int64 f3g2    = f3   * (crypto_int64) g2;
  crypto_int64 f3g3_2  = f3_2 * (crypto_int64) g3;
  crypto_int64 f3g4    = f3   * (crypto_int64) g4;
  crypto_int64 f3g5_2  = f3_2 * (crypto_int64) g5;
  crypto_int64 f3g6    = f3   * (crypto_int64) g6;
  crypto_int64 f3g7_38 = f3_2 * (crypto_int64) g7_19;
  crypto_int64 f3g8_19 = f3   * (crypto_int64) g8_19;
  crypto_int64 f3g9_38 = f3_2 * (crypto_int64) g9_19;
  crypto_int64 f4g0    = f4   * (crypto_int64) g0;
  crypto_int64 f4g1    = f4   * (crypto_int64) g1;
  crypto_int64 f4g2    = f4   * (crypto_int64) g2;
  crypto_int64 f4g3    = f4   * (crypto_int64) g3;
  crypto_int64 f4g4    = f4   * (crypto_int64) g4;
  crypto_int64 f4g5    = f4   * (crypto_int64) g5;
  crypto_int64 f4g6_19 = f4   * (crypto_int64) g6_19;
  crypto_int64 f4g7_19 = f4   * (crypto_int64) g7_19;
  crypto_int64 f4g8_19 = f4   * (crypto_int64) g8_19;
  crypto_int64 f4g9_19 = f4   * (crypto_int64) g9_19;
  crypto_int64 f5g0    = f5   * (crypto_int64) g0;
  crypto_int64 f5g1_2  = f5_2 * (crypto_int64) g1;
  crypto_int64 f5g2    = f5   * (crypto_int64) g2;
  crypto_int64 f5g3_2  = f5_2 * (crypto_int64) g3;
  crypto_int64 f5g4    = f5   * (crypto_int64) g4;
  crypto_int64 f5g5_38 = f5_2 * (crypto_int64) g5_19;
  crypto_int64 f5g6_19 = f5   * (crypto_int64) g6_19;
  crypto_int64 f5g7_38 = f5_2 * (crypto_int64) g7_19;
  crypto_int64 f5g8_19 = f5   * (crypto_int64) g8_19;
  crypto_int64 f5g9_38 = f5_2 * (crypto_int64) g9_19;
  crypto_int64 f6g0    = f6   * (crypto_int64) g0;
  crypto_int64 f6g1    = f6   * (crypto_int64) g1;
  crypto_int64 f6g2    = f6   * (crypto_int64) g2;
  crypto_int64 f6g3    = f6   * (crypto_int64) g3;
  crypto_int64 f6g4_19 = f6   * (crypto_int64) g4_19;
  crypto_int64 f6g5_19 = f6   * (crypto_int64) g5_19;
  crypto_int64 f6g6_19 = f6   * (crypto_int64) g6_19;
  crypto_int64 f6g7_19 = f6   * (crypto_int64) g7_19;
  crypto_int64 f6g8_19 = f6   * (crypto_int64) g8_19;
  crypto_int64 f6g9_19 = f6   * (crypto_int64) g9_19;
  crypto_int64 f7g0    = f7   * (crypto_int64) g0;
  crypto_int64 f7g1_2  = f7_2 * (crypto_int64) g1;
  crypto_int64 f7g2    = f7   * (crypto_int64) g2;
  crypto_int64 f7g3_38 = f7_2 * (crypto_int64) g3_19;
  crypto_int64 f7g4_19 = f7   * (crypto_int64) g4_19;
  crypto_int64 f7g5_38 = f7_2 * (crypto_int64) g5_19;
  crypto_int64 f7g6_19 = f7   * (crypto_int64) g6_19;
  crypto_int64 f7g7_38 = f7_2 * (crypto_int64) g7_19;
  crypto_int64 f7g8_19 = f7   * (crypto_int64) g8_19;
  crypto_int64 f7g9_38 = f7_2 * (crypto_int64) g9_19;
  crypto_int64 f8g0    = f8   * (crypto_int64) g0;
  crypto_int64 f8g1    = f8   * (crypto_int64) g1;
  crypto_int64 f8g2_19 = f8   * (crypto_int64) g2_19;
  crypto_int64 f8g3_19 = f8   * (crypto_int64) g3_19;
  crypto_int64 f8g4_19 = f8   * (crypto_int64) g4_19;
  crypto_int64 f8g5_19 = f8   * (crypto_int64) g5_19;
  crypto_int64 f8g6_19 = f8   * (crypto_int64) g6_19;
  crypto_int64 f8g7_19 = f8   * (crypto_int64) g7_19;
  crypto_int64 f8g8_19 = f8   * (crypto_int64) g8_19;
  crypto_int64 f8g9_19 = f8   * (crypto_int64) g9_19;
  crypto_int64 f9g0    = f9   * (crypto_int64) g0;
  crypto_int64 f9g1_38 = f9_2 * (crypto_int64) g1_19;
  crypto_int64 f9g2_19 = f9   * (crypto_int64) g2_19;
  crypto_int64 f9g3_38 = f9_2 * (crypto_int64) g3_19;
  crypto_int64 f9g4_19 = f9   * (crypto_int64) g4_19;
  crypto_int64 f9g5_38 = f9_2 * (crypto_int64) g5_19;
  crypto_int64 f9g6_19 = f9   * (crypto_int64) g6_19;
  crypto_int64 f9g7_38 = f9_2 * (crypto_int64) g7_19;
  crypto_int64 f9g8_19 = f9   * (crypto_int64) g8_19;
  crypto_int64 f9g9_38 = f9_2 * (crypto_int64) g9_19;
  crypto_int64 h0 = f0g0+f1g9_38+f2g8_19+f3g7_38+f4g6_19+f5g5_38+f6g4_19+f7g3_38+f8g2_19+f9g1_38;
  crypto_int64 h1 = f0g1+f1g0   +f2g9_19+f3g8_19+f4g7_19+f5g6_19+f6g5_19+f7g4_19+f8g3_19+f9g2_19;
  crypto_int64 h2 = f0g2+f1g1_2 +f2g0   +f3g9_38+f4g8_19+f5g7_38+f6g6_19+f7g5_38+f8g4_19+f9g3_38;
  crypto_int64 h3 = f0g3+f1g2   +f2g1   +f3g0   +f4g9_19+f5g8_19+f6g7_19+f7g6_19+f8g5_19+f9g4_19;
  crypto_int64 h4 = f0g4+f1g3_2 +f2g2   +f3g1_2 +f4g0   +f5g9_38+f6g8_19+f7g7_38+f8g6_19+f9g5_38;
  crypto_int64 h5 = f0g5+f1g4   +f2g3   +f3g2   +f4g1   +f5g0   +f6g9_19+f7g8_19+f8g7_19+f9g6_19;
  crypto_int64 h6 = f0g6+f1g5_2 +f2g4   +f3g3_2 +f4g2   +f5g1_2 +f6g0   +f7g9_38+f8g8_19+f9g7_38;
  crypto_int64 h7 = f0g7+f1g6   +f2g5   +f3g4   +f4g3   +f5g2   +f6g1   +f7g0   +f8g9_19+f9g8_19;
  crypto_int64 h8 = f0g8+f1g7_2 +f2g6   +f3g5_2 +f4g4   +f5g3_2 +f6g2   +f7g1_2 +f8g0   +f9g9_38;
  crypto_int64 h9 = f0g9+f1g8   +f2g7   +f3g6   +f4g5   +f5g4   +f6g3   +f7g2   +f8g1   +f9g0   ;
  crypto_int64 carry0;
  crypto_int64 carry1;
  crypto_int64 carry2;
  crypto_int64 carry3;
  crypto_int64 carry4;
  crypto_int64 carry5;
  crypto_int64 carry6;
  crypto_int64 carry7;
  crypto_int64 carry8;
  crypto_int64 carry9;
  
  /*
   *  |h0| <= (1.65*1.65*2^52*(1+19+19+19+19)+1.65*1.65*2^50*(38+38+38+38+38))
   *    i.e. |h0| <= 1.4*2^60; narrower ranges for h2, h4, h6, h8
   *  |h1| <= (1.65*1.65*2^51*(1+1+19+19+19+19+19+19+19+19))
   *    i.e. |h1| <= 1.7*2^59; narrower ranges for h3, h5, h7, h9
   */
  
  carry0 = (h0 + (crypto_int64) (1<<25)) >> 26; h1 += carry0; h0 -= carry0 << 26;
  carry4 = (h4 + (crypto_int64) (1<<25)) >> 26; h5 += carry4; h4 -= carry4 << 26;
  /* |h0| <= 2^25 */
  /* |h4| <= 2^25 */
  /* |h1| <= 1.71*2^59 */
  /* |h5| <= 1.71*2^59 */
  
  carry1 = (h1 + (crypto_int64) (1<<24)) >> 25; h2 += carry1; h1 -= carry1 << 25;
  carry5 = (h5 + (crypto_int64) (1<<24)) >> 25; h6 += carry5; h5 -= carry5 << 25;
  /* |h1| <= 2^24; from now on fits into int32 */
  /* |h5| <= 2^24; from now on fits into int32 */
  /* |h2| <= 1.41*2^60 */
  /* |h6| <= 1.41*2^60 */
  
  carry2 = (h2 + (crypto_int64) (1<<25)) >> 26; h3 += carry2; h2 -= carry2 << 26;
  carry6 = (h6 + (crypto_int64) (1<<25)) >> 26; h7 += carry6; h6 -= carry6 << 26;
  /* |h2| <= 2^25; from now on fits into int32 unchanged */
  /* |h6| <= 2^25; from now on fits into int32 unchanged */
  /* |h3| <= 1.71*2^59 */
  /* |h7| <= 1.71*2^59 */
  
  carry3 = (h3 + (crypto_int64) (1<<24)) >> 25; h4 += carry3; h3 -= carry3 << 25;
  carry7 = (h7 + (crypto_int64) (1<<24)) >> 25; h8 += carry7; h7 -= carry7 << 25;
  /* |h3| <= 2^24; from now on fits into int32 unchanged */
  /* |h7| <= 2^24; from now on fits into int32 unchanged */
  /* |h4| <= 1.72*2^34 */
  /* |h8| <= 1.41*2^60 */
  
  carry4 = (h4 + (crypto_int64) (1<<25)) >> 26; h5 += carry4; h4 -= carry4 << 26;
  carry8 = (h8 + (crypto_int64) (1<<25)) >> 26; h9 += carry8; h8 -= carry8 << 26;
  /* |h4| <= 2^25; from now on fits into int32 unchanged */
  /* |h8| <= 2^25; from now on fits into int32 unchanged */
  /* |h5| <= 1.01*2^24 */
  /* |h9| <= 1.71*2^59 */
  
  carry9 = (h9 + (crypto_int64) (1<<24)) >> 25; h0 += carry9 * 19; h9 -= carry9 << 25;
  /* |h9| <= 2^24; from now on fits into int32 unchanged */
  /* |h0| <= 1.1*2^39 */
  
  carry0 = (h0 + (crypto_int64) (1<<25)) >> 26; h1 += carry0; h0 -= carry0 << 26;
  /* |h0| <= 2^25; from now on fits into int32 unchanged */
  /* |h1| <= 1.01*2^24 */
  
  h->v[0] = h0;
  h->v[1] = h1;
  h->v[2] = h2;
  h->v[3] = h3;
  h->v[4] = h4;
  h->v[5] = h5;
  h->v[6] = h6;
  h->v[7] = h7;
  h->v[8] = h8;
  h->v[9] = h9;
}

/*
 * h = f * f
 * Can overlap h with f.
 * 
 * Preconditions:
 *   |f| bounded by 1.65*2^26,1.65*2^25,1.65*2^26,1.65*2^25,etc.
 * 
 * Postconditions:
 *   |h| bounded by 1.01*2^25,1.01*2^24,1.01*2^25,1.01*2^24,etc.
 */
void fe25519_square(fe25519 *h,const fe25519 *f)
{
  crypto_int32 f0 = f->v[0];
  crypto_int32 f1 = f->v[1];
  crypto_int32 f2 = f->v[2];
  crypto_int32 f3 = f->v[3];
  crypto_int32 f4 = f->v[4];
  crypto_int32 f5 = f->v[5];
  crypto_int32 f6 = f->v[6];
  crypto_int32 f7 = f->v[7];
  crypto_int32 f8 = f->v[8];
  crypto_int32 f9 = f->v[9];
  crypto_int32 f0_2 = 2 * f0;
  crypto_int32 f1_2 = 2 * f1;
  crypto_int32 f2_2 = 2 * f2;
  crypto_int32 f3_2 = 2 * f3;
  crypto_int32 f4_2 = 2 * f4;
  crypto_int32 f5_2 = 2 * f5;
  crypto_int32 f6_2 = 2 * f6;
  crypto_int32 f7_2 = 2 * f7;
  crypto_int32 f5_38 = 38 * f5; /* 1.959375*2^30 */
  crypto_int32 f6_19 = 19 * f6; /* 1.959375*2^30 */
  crypto_int32 f7_38 = 38 * f7; /* 1.959375*2^30 */
  crypto_int32 f8_19 = 19 * f8; /* 1.959375*2^30 */
  crypto_int32 f9_38 = 38 * f9; /* 1.959375*2^30 */
  crypto_int64 f0f0    = f0   * (crypto_int64) f0;
  crypto_int64 f0f1_2  = f0_2 * (crypto_int64) f1;
  crypto_int64 f0f2_2  = f0_2 * (crypto_int64) f2;
  crypto_int64 f0f3_2  = f0_2 * (crypto_int64) f3;
  crypto_int64 f0f4_2  = f0_2 * (crypto_int64) f4;
  crypto_int64 f0f5_2  = f0_2 * (crypto_int64) f5;
  crypto_int64 f0f6_2  = f0_2 * (crypto_int64) f6;
  crypto_int64 f0f7_2  = f0_2 * (crypto_int64) f7;
  crypto_int64 f0f8_2  = f0_2 * (crypto_int64) f8;
  crypto_int64 f0f9_2  = f0_2 * (crypto_int64) f9;
  crypto_int64 f1f1_2  = f1_2 * (crypto_int64) f1;
  crypto_int64 f1f2_2  = f1_2 * (crypto_int64) f2;
  crypto_int64 f1f3_4  = f1_2 * (crypto_int64) f3_2;
  crypto_int64 f1f4_2  = f1_2 * (crypto_int64) f4;
  crypto_int64 f1f5_4  = f1_2 * (crypto_int64) f5_2;
  crypto_int64 f1f6_2  = f1_2 * (crypto_int64) f6;
  crypto_int64 f1f7_4  = f1_2 * (crypto_int64) f7_2;
  crypto_int64 f1f8_2  = f1_2 * (crypto_int64) f8;
  crypto_int64 f1f9_76 = f1_2 * (crypto_int64) f9_38;
  crypto_int64 f2f2    = f2   * (crypto_int64) f2;
  crypto_int64 f2f3_2  = f2_2 * (crypto_int64) f3;
  crypto_int64 f2f4_2  = f2_2 * (crypto_int64) f4;
  crypto_int64 f2f5_2  = f2_2 * (crypto_int64) f5;
  crypto_int64 f2f6_2  = f2_2 * (crypto_int64) f6;
  crypto_int64 f2f7_2  = f2_2 * (crypto_int64) f7;
  crypto_int64 f2f8_38 = f2_2 * (crypto_int64) f8_19;
  crypto_int64 f2f9_38 = f2   * (crypto_int64) f9_38;
  crypto_int64 f3f3_2  = f3_2 * (crypto_int64) f3;
  crypto_int64 f3f4_2  = f3_2 * (crypto_int64) f4;
  crypto_int64 f3f5_4  = f3_2 * (crypto_int64) f5_2;
  crypto_int64 f3f6_2  = f3_2 * (crypto_int64) f6;
  crypto_int64 f3f7_76 = f3_2 * (crypto_int64) f7_38;
  crypto_int64 f3f8_38 = f3_2 * (crypto_int64) f8_19;
  crypto_int64 f3f9_76 = f3_2 * (crypto_int64) f9_38;
  crypto_int64 f4f4    = f4   * (crypto_int64) f4;
  crypto_int64 f4f5_2  = f4_2 * (crypto_int64) f5;
  crypto_int64 f4f6_38 = f4_2 * (crypto_int64) f6_19;
  crypto_int64 f4f7_38 = f4   * (crypto_int64) f7_38;
  crypto_int64 f4f8_38 = f4_2 * (crypto_int64) f8_19;
  crypto_int64 f4f9_38 = f4   * (crypto_int64) f9_38;
  crypto_int64 f5f5_38 = f5   * (crypto_int64) f5_38;
  crypto_int64 f5f6_38 = f5_2 * (crypto_int64) f6_19;
  crypto_int64 f5f7_76 = f5_2 * (crypto_int64) f7_38;
  crypto_int64 f5f8_38 = f5_2 * (crypto_int64) f8_19;
  crypto_int64 f5f9_76 = f5_2 * (crypto_int64) f9_38;
  crypto_int64 f6f6_19 = f6   * (crypto_int64) f6_19;
  crypto_int64 f6f7_38 = f6   * (crypto_int64) f7_38;
  crypto_int64 f6f8_38 = f6_2 * (crypto_int64) f8_19;
  crypto_int64 f6f9_38 = f6   * (crypto_int64) f9_38;
  crypto_int64 f7f7_38 = f7   * (crypto_int64) f7_38;
  crypto_int64 f7f8_38 = f7_2 * (crypto_int64) f8_19;
  crypto_int64 f7f9_76 = f7_2 * (crypto_int64) f9_38;
  crypto_int64 f8f8_19 = f8   * (crypto_int64) f8_19;
  crypto_int64 f8f9_38 = f8   * (crypto_int64) f9_38;
  crypto_int64 f9f9_38 = f9   * (crypto_int64) f9_38;
  crypto_int64 h0 = f0f0  +f1f9_76+f2f8_38+f3f7_76+f4f6_38+f5f5_38;
  crypto_int64 h1 = f0f1_2+f2f9_38+f3f8_38+f4f7_38+f5f6_38;
  crypto_int64 h2 = f0f2_2+f1f1_2 +f3f9_76+f4f8_38+f5f7_76+f6f6_19;
  crypto_int64 h3 = f0f3_2+f1f2_2 +f4f9_38+f5f8_38+f6f7_38;
  crypto_int64 h4 = f0f4_2+f1f3_4 +f2f2   +f5f9_76+f6f8_38+f7f7_38;
  crypto_int64 h5 = f0f5_2+f1f4_2 +f2f3_2 +f6f9_38+f7f8_38;
  crypto_int64 h6 = f0f6_2+f1f5_4 +f2f4_2 +f3f3_2 +f7f9_76+f8f8_19;
  crypto_int64 h7 = f0f7_2+f1f6_2 +f2f5_2 +f3f4_2 +f8f9_38;
  crypto_int64 h8 = f0f8_2+f1f7_4 +f2f6_2 +f3f5_4 +f4f4   +f9f9_38;
  crypto_int64 h9 = f0f9_2+f1f8_2 +f2f7_2 +f3f6_2 +f4f5_2;
  crypto_int64 carry0;
  crypto_int64 carry1;
  crypto_int64 carry2;
  crypto_int64 carry3;
  crypto_int64 carry4;
  crypto_int64 carry5;
  crypto_int64 carry6;
  crypto_int64 carry7;
  crypto_int64 carry8;
  crypto_int64 carry9;
  
  carry0 = (h0 + (crypto_int64) (1<<25)) >> 26; h1 += carry0; h0 -= carry0 << 26;
  carry4 = (h4 + (crypto_int64) (1<<25)) >> 26; h5 += carry4; h4 -= carry4 << 26;
  
  carry1 = (h1 + (crypto_int64) (1<<24)) >> 25; h2 += carry1; h1 -= carry1 << 25;
  carry5 = (h5 + (crypto_int64) (1<<24)) >> 25; h6 += carry5; h5 -= carry5 << 25;
  
  carry2 = (h2 + (crypto_int64) (1<<25)) >> 26; h3 += carry2; h2 -= carry2 << 26;
  carry6 = (h6 + (crypto_int64) (1<<25)) >> 26; h7 += carry6; h6 -= carry6 << 26;
  
  carry3 = (h3 + (crypto_int64) (1<<24)) >> 25; h4 += carry3; h3 -= carry3 << 25;
  carry7 = (h7 + (crypto_int64) (1<<24)) >> 25; h8 += carry7; h7 -= carry7 << 25;
  
  carry4 = (h4 + (crypto_int64) (1<<25)) >> 26; h5 += carry4; h4 -= carry4 << 26;
  carry8 = (h8 + (crypto_int64) (1<<25)) >> 26; h9 += carry8; h8 -= carry8 << 26;
  
  carry9 = (h9 + (crypto_int64) (1<<24)) >> 25; h0 += carry9 * 19; h9 -= carry9 << 25;
  
  carry0 = (h0 + (crypto_int64) (1<<25)) >> 26; h1 += carry0; h0 -= carry0 << 26;
  
  h->v[0] = h0;
  h->v[1] = h1;
  h->v[2] = h2;
  h->v[3] = h3;
  h->v[4] = h4;
  h->v[5] = h5;
  h->v[6] = h6;
  h->v[7] = h7;
  h->v[8] = h8;
  h->v[9] = h9;
}


void fe25519_invert(fe25519 *out,const fe25519 *z)
{
  fe25519 t0;
  fe25519 t1;
  fe25519 t2;
  fe25519 t3;
  int i;
  
  /* qhasm: fe z1 */
  
  /* qhasm: fe z2 */
  
  /* qhasm: fe z8 */
  
  /* qhasm: fe z9 */
  
  /* qhasm: fe z11 */
  
  /* qhasm: fe z22 */
  
  /* qhasm: fe z_5_0 */
  
  /* qhasm: fe z_10_5 */
  
  /* qhasm: fe z_10_0 */
  
  /* qhasm: fe z_20_10 */
  
  /* qhasm: fe z_20_0 */
  
  /* qhasm: fe z_40_20 */
  
  /* qhasm: fe z_40_0 */
  
  /* qhasm: fe z_50_10 */
  
  /* qhasm: fe z_50_0 */
  
  /* qhasm: fe z_100_50 */
  
  /* qhasm: fe z_100_0 */
  
  /* qhasm: fe z_200_100 */
  
  /* qhasm: fe z_200_0 */
  
  /* qhasm: fe z_250_50 */
  
  /* qhasm: fe z_250_0 */
  
  /* qhasm: fe z_255_5 */
  
  /* qhasm: fe z_255_21 */
  
  /* qhasm: enter pow225521 */
  
  /* qhasm: z2 = z1^2^1 */
  /* asm 1: fe25519_square(>z2=fe#1,<z1=fe#11); for (i = 1;i < 1;++i) fe25519_square(>z2=fe#1,>z2=fe#1); */
  /* asm 2: fe25519_square(>z2=&t0,<z1=z); for (i = 1;i < 1;++i) fe25519_square(>z2=&t0,>z2=&t0); */
  fe25519_square(&t0,z); for (i = 1;i < 1;++i) fe25519_square(&t0,&t0);
  
  /* qhasm: z8 = z2^2^2 */
  /* asm 1: fe25519_square(>z8=fe#2,<z2=fe#1); for (i = 1;i < 2;++i) fe25519_square(>z8=fe#2,>z8=fe#2); */
  /* asm 2: fe25519_square(>z8=&t1,<z2=&t0); for (i = 1;i < 2;++i) fe25519_square(>z8=&t1,>z8=&t1); */
  fe25519_square(&t1,&t0); for (i = 1;i < 2;++i) fe25519_square(&t1,&t1);
  
  /* qhasm: z9 = z1*z8 */
  /* asm 1: fe25519_mul(>z9=fe#2,<z1=fe#11,<z8=fe#2); */
  /* asm 2: fe25519_mul(>z9=&t1,<z1=z,<z8=&t1); */
  fe25519_mul(&t1,z,&t1);
  
  /* qhasm: z11 = z2*z9 */
  /* asm 1: fe25519_mul(>z11=fe#1,<z2=fe#1,<z9=fe#2); */
  /* asm 2: fe25519_mul(>z11=&t0,<z2=&t0,<z9=&t1); */
  fe25519_mul(&t0,&t0,&t1);
  
  /* qhasm: z22 = z11^2^1 */
  /* asm 1: fe25519_square(>z22=fe#3,<z11=fe#1); for (i = 1;i < 1;++i) fe25519_square(>z22=fe#3,>z22=fe#3); */
  /* asm 2: fe25519_square(>z22=&t2,<z11=&t0); for (i = 1;i < 1;++i) fe25519_square(>z22=&t2,>z22=&t2); */
  fe25519_square(&t2,&t0); for (i = 1;i < 1;++i) fe25519_square(&t2,&t2);
  
  /* qhasm: z_5_0 = z9*z22 */
  /* asm 1: fe25519_mul(>z_5_0=fe#2,<z9=fe#2,<z22=fe#3); */
  /* asm 2: fe25519_mul(>z_5_0=&t1,<z9=&t1,<z22=&t2); */
  fe25519_mul(&t1,&t1,&t2);
  
  /* qhasm: z_10_5 = z_5_0^2^5 */
  /* asm 1: fe25519_square(>z_10_5=fe#3,<z_5_0=fe#2); for (i = 1;i < 5;++i) fe25519_square(>z_10_5=fe#3,>z_10_5=fe#3); */
  /* asm 2: fe25519_square(>z_10_5=&t2,<z_5_0=&t1); for (i = 1;i < 5;++i) fe25519_square(>z_10_5=&t2,>z_10_5=&t2); */
  fe25519_square(&t2,&t1); for (i = 1;i < 5;++i) fe25519_square(&t2,&t2);
  
  /* qhasm: z_10_0 = z_10_5*z_5_0 */
  /* asm 1: fe25519_mul(>z_10_0=fe#2,<z_10_5=fe#3,<z_5_0=fe#2); */
  /* asm 2: fe25519_mul(>z_10_0=&t1,<z_10_5=&t2,<z_5_0=&t1); */
  fe25519_mul(&t1,&t2,&t1);
  
  /* qhasm: z_20_10 = z_10_0^2^10 */
  /* asm 1: fe25519_square(>z_20_10=fe#3,<z_10_0=fe#2); for (i = 1;i < 10;++i) fe25519_square(>z_20_10=fe#3,>z_20_10=fe#3); */
  /* asm 2: fe25519_square(>z_20_10=&t2,<z_10_0=&t1); for (i = 1;i < 10;++i) fe25519_square(>z_20_10=&t2,>z_20_10=&t2); */
  fe25519_square(&t2,&t1); for (i = 1;i < 10;++i) fe25519_square(&t2,&t2);
  
  /* qhasm: z_20_0 = z_20_10*z_10_0 */
  /* asm 1: fe25519_mul(>z_20_0=fe#3,<z_20_10=fe#3,<z_10_0=fe#2); */
  /* asm 2: fe25519_mul(>z_20_0=&t2,<z_20_10=&t2,<z_10_0=&t1); */
  fe25519_mul(&t2,&t2,&t1);
  
  /* qhasm: z_40_20 = z_20_0^2^20 */
  /* asm 1: fe25519_square(>z_40_20=fe#4,<z_20_0=fe#3); for (i = 1;i < 20;++i) fe25519_square(>z_40_20=fe#4,>z_40_20=fe#4); */
  /* asm 2: fe25519_square(>z_40_20=&t3,<z_20_0=&t2); for (i = 1;i < 20;++i) fe25519_square(>z_40_20=&t3,>z_40_20=&t3); */
  fe25519_square(&t3,&t2); for (i = 1;i < 20;++i) fe25519_square(&t3,&t3);
  
  /* qhasm: z_40_0 = z_40_20*z_20_0 */
  /* asm 1: fe25519_mul(>z_40_0=fe#3,<z_40_20=fe#4,<z_20_0=fe#3); */
  /* asm 2: fe25519_mul(>z_40_0=&t2,<z_40_20=&t3,<z_20_0=&t2); */
  fe25519_mul(&t2,&t3,&t2);
  
  /* qhasm: z_50_10 = z_40_0^2^10 */
  /* asm 1: fe25519_square(>z_50_10=fe#3,<z_40_0=fe#3); for (i = 1;i < 10;++i) fe25519_square(>z_50_10=fe#3,>z_50_10=fe#3); */
  /* asm 2: fe25519_square(>z_50_10=&t2,<z_40_0=&t2); for (i = 1;i < 10;++i) fe25519_square(>z_50_10=&t2,>z_50_10=&t2); */
  fe25519_square(&t2,&t2); for (i = 1;i < 10;++i) fe25519_square(&t2,&t2);
  
  /* qhasm: z_50_0 = z_50_10*z_10_0 */
  /* asm 1: fe25519_mul(>z_50_0=fe#2,<z_50_10=fe#3,<z_10_0=fe#2); */
  /* asm 2: fe25519_mul(>z_50_0=&t1,<z_50_10=&t2,<z_10_0=&t1); */
  fe25519_mul(&t1,&t2,&t1);
  
  /* qhasm: z_100_50 = z_50_0^2^50 */
  /* asm 1: fe25519_square(>z_100_50=fe#3,<z_50_0=fe#2); for (i = 1;i < 50;++i) fe25519_square(>z_100_50=fe#3,>z_100_50=fe#3); */
  /* asm 2: fe25519_square(>z_100_50=&t2,<z_50_0=&t1); for (i = 1;i < 50;++i) fe25519_square(>z_100_50=&t2,>z_100_50=&t2); */
  fe25519_square(&t2,&t1); for (i = 1;i < 50;++i) fe25519_square(&t2,&t2);
  
  /* qhasm: z_100_0 = z_100_50*z_50_0 */
  /* asm 1: fe25519_mul(>z_100_0=fe#3,<z_100_50=fe#3,<z_50_0=fe#2); */
  /* asm 2: fe25519_mul(>z_100_0=&t2,<z_100_50=&t2,<z_50_0=&t1); */
  fe25519_mul(&t2,&t2,&t1);
  
  /* qhasm: z_200_100 = z_100_0^2^100 */
  /* asm 1: fe25519_square(>z_200_100=fe#4,<z_100_0=fe#3); for (i = 1;i < 100;++i) fe25519_square(>z_200_100=fe#4,>z_200_100=fe#4); */
  /* asm 2: fe25519_square(>z_200_100=&t3,<z_100_0=&t2); for (i = 1;i < 100;++i) fe25519_square(>z_200_100=&t3,>z_200_100=&t3); */
  fe25519_square(&t3,&t2); for (i = 1;i < 100;++i) fe25519_square(&t3,&t3);
  
  /* qhasm: z_200_0 = z_200_100*z_100_0 */
  /* asm 1: fe25519_mul(>z_200_0=fe#3,<z_200_100=fe#4,<z_100_0=fe#3); */
  /* asm 2: fe25519_mul(>z_200_0=&t2,<z_200_100=&t3,<z_100_0=&t2); */
  fe25519_mul(&t2,&t3,&t2);
  
  /* qhasm: z_250_50 = z_200_0^2^50 */
  /* asm 1: fe25519_square(>z_250_50=fe#3,<z_200_0=fe#3); for (i = 1;i < 50;++i) fe25519_square(>z_250_50=fe#3,>z_250_50=fe#3); */
  /* asm 2: fe25519_square(>z_250_50=&t2,<z_200_0=&t2); for (i = 1;i < 50;++i) fe25519_square(>z_250_50=&t2,>z_250_50=&t2); */
  fe25519_square(&t2,&t2); for (i = 1;i < 50;++i) fe25519_square(&t2,&t2);
  
  /* qhasm: z_250_0 = z_250_50*z_50_0 */
  /* asm 1: fe25519_mul(>z_250_0=fe#2,<z_250_50=fe#3,<z_50_0=fe#2); */
  /* asm 2: fe25519_mul(>z_250_0=&t1,<z_250_50=&t2,<z_50_0=&t1); */
  fe25519_mul(&t1,&t2,&t1);
  
  /* qhasm: z_255_5 = z_250_0^2^5 */
  /* asm 1: fe25519_square(>z_255_5=fe#2,<z_250_0=fe#2); for (i = 1;i < 5;++i) fe25519_square(>z_255_5=fe#2,>z_255_5=fe#2); */
  /* asm 2: fe25519_square(>z_255_5=&t1,<z_250_0=&t1); for (i = 1;i < 5;++i) fe25519_square(>z_255_5=&t1,>z_255_5=&t1); */
  fe25519_square(&t1,&t1); for (i = 1;i < 5;++i) fe25519_square(&t1,&t1);
  
  /* qhasm: z_255_21 = z_255_5*z11 */
  /* asm 1: fe25519_mul(>z_255_21=fe#12,<z_255_5=fe#2,<z11=fe#1); */
  /* asm 2: fe25519_mul(>z_255_21=out,<z_255_5=&t1,<z11=&t0); */
  fe25519_mul(out,&t1,&t0);
  
  /* qhasm: return */
  
  return;
}


void fe25519_pow2523(fe25519 *out,const fe25519 *z)
{
  fe25519 t0;
  fe25519 t1;
  fe25519 t2;
  int i;
  
  /* qhasm: fe z1 */
  
  /* qhasm: fe z2 */
  
  /* qhasm: fe z8 */
  
  /* qhasm: fe z9 */
  
  /* qhasm: fe z11 */
  
  /* qhasm: fe z22 */
  
  /* qhasm: fe z_5_0 */
  
  /* qhasm: fe z_10_5 */
  
  /* qhasm: fe z_10_0 */
  
  /* qhasm: fe z_20_10 */
  
  /* qhasm: fe z_20_0 */
  
  /* qhasm: fe z_40_20 */
  
  /* qhasm: fe z_40_0 */
  
  /* qhasm: fe z_50_10 */
  
  /* qhasm: fe z_50_0 */
  
  /* qhasm: fe z_100_50 */
  
  /* qhasm: fe z_100_0 */
  
  /* qhasm: fe z_200_100 */
  
  /* qhasm: fe z_200_0 */
  
  /* qhasm: fe z_250_50 */
  
  /* qhasm: fe z_250_0 */
  
  /* qhasm: fe z_252_2 */
  
  /* qhasm: fe z_252_3 */
  
  /* qhasm: enter pow22523 */
  
  /* qhasm: z2 = z1^2^1 */
  /* asm 1: fe25519_square(>z2=fe#1,<z1=fe#11); for (i = 1;i < 1;++i) fe25519_square(>z2=fe#1,>z2=fe#1); */
  /* asm 2: fe25519_square(>z2=&t0,<z1=z); for (i = 1;i < 1;++i) fe25519_square(>z2=&t0,>z2=&t0); */
  fe25519_square(&t0,z); for (i = 1;i < 1;++i) fe25519_square(&t0,&t0);
  
  /* qhasm: z8 = z2^2^2 */
  /* asm 1: fe25519_square(>z8=fe#2,<z2=fe#1); for (i = 1;i < 2;++i) fe25519_square(>z8=fe#2,>z8=fe#2); */
  /* asm 2: fe25519_square(>z8=&t1,<z2=&t0); for (i = 1;i < 2;++i) fe25519_square(>z8=&t1,>z8=&t1); */
  fe25519_square(&t1,&t0); for (i = 1;i < 2;++i) fe25519_square(&t1,&t1);
  
  /* qhasm: z9 = z1*z8 */
  /* asm 1: fe25519_mul(>z9=fe#2,<z1=fe#11,<z8=fe#2); */
  /* asm 2: fe25519_mul(>z9=&t1,<z1=z,<z8=&t1); */
  fe25519_mul(&t1,z,&t1);
  
  /* qhasm: z11 = z2*z9 */
  /* asm 1: fe25519_mul(>z11=fe#1,<z2=fe#1,<z9=fe#2); */
  /* asm 2: fe25519_mul(>z11=&t0,<z2=&t0,<z9=&t1); */
  fe25519_mul(&t0,&t0,&t1);
  
  /* qhasm: z22 = z11^2^1 */
  /* asm 1: fe25519_square(>z22=fe#1,<z11=fe#1); for (i = 1;i < 1;++i) fe25519_square(>z22=fe#1,>z22=fe#1); */
  /* asm 2: fe25519_square(>z22=&t0,<z11=&t0); for (i = 1;i < 1;++i) fe25519_square(>z22=&t0,>z22=&t0); */
  fe25519_square(&t0,&t0); for (i = 1;i < 1;++i) fe25519_square(&t0,&t0);
  
  /* qhasm: z_5_0 = z9*z22 */
  /* asm 1: fe25519_mul(>z_5_0=fe#1,<z9=fe#2,<z22=fe#1); */
  /* asm 2: fe25519_mul(>z_5_0=&t0,<z9=&t1,<z22=&t0); */
  fe25519_mul(&t0,&t1,&t0);
  
  /* qhasm: z_10_5 = z_5_0^2^5 */
  /* asm 1: fe25519_square(>z_10_5=fe#2,<z_5_0=fe#1); for (i = 1;i < 5;++i) fe25519_square(>z_10_5=fe#2,>z_10_5=fe#2); */
  /* asm 2: fe25519_square(>z_10_5=&t1,<z_5_0=&t0); for (i = 1;i < 5;++i) fe25519_square(>z_10_5=&t1,>z_10_5=&t1); */
  fe25519_square(&t1,&t0); for (i = 1;i < 5;++i) fe25519_square(&t1,&t1);
  
  /* qhasm: z_10_0 = z_10_5*z_5_0 */
  /* asm 1: fe25519_mul(>z_10_0=fe#1,<z_10_5=fe#2,<z_5_0=fe#1); */
  /* asm 2: fe25519_mul(>z_10_0=&t0,<z_10_5=&t1,<z_5_0=&t0); */
  fe25519_mul(&t0,&t1,&t0);
  
  /* qhasm: z_20_10 = z_10_0^2^10 */
  /* asm 1: fe25519_square(>z_20_10=fe#2,<z_10_0=fe#1); for (i = 1;i < 10;++i) fe25519_square(>z_20_10=fe#2,>z_20_10=fe#2); */
  /* asm 2: fe25519_square(>z_20_10=&t1,<z_10_0=&t0); for (i = 1;i < 10;++i) fe25519_square(>z_20_10=&t1,>z_20_10=&t1); */
  fe25519_square(&t1,&t0); for (i = 1;i < 10;++i) fe25519_square(&t1,&t1);
  
  /* qhasm: z_20_0 = z_20_10*z_10_0 */
  /* asm 1: fe25519_mul(>z_20_0=fe#2,<z_20_10=fe#2,<z_10_0=fe#1); */
  /* asm 2: fe25519_mul(>z_20_0=&t1,<z_20_10=&t1,<z_10_0=&t0); */
  fe25519_mul(&t1,&t1,&t0);
  
  /* qhasm: z_40_20 = z_20_0^2^20 */
  /* asm 1: fe25519_square(>z_40_20=fe#3,<z_20_0=fe#2); for (i = 1;i < 20;++i) fe25519_square(>z_40_20=fe#3,>z_40_20=fe#3); */
  /* asm 2: fe25519_square(>z_40_20=&t2,<z_20_0=&t1); for (i = 1;i < 20;++i) fe25519_square(>z_40_20=&t2,>z_40_20=&t2); */
  fe25519_square(&t2,&t1); for (i = 1;i < 20;++i) fe25519_square(&t2,&t2);
  
  /* qhasm: z_40_0 = z_40_20*z_20_0 */
  /* asm 1: fe25519_mul(>z_40_0=fe#2,<z_40_20=fe#3,<z_20_0=fe#2); */
  /* asm 2: fe25519_mul(>z_40_0=&t1,<z_40_20=&t2,<z_20_0=&t1); */
  fe25519_mul(&t1,&t2,&t1);
  
  /* qhasm: z_50_10 = z_40_0^2^10 */
  /* asm 1: fe25519_square(>z_50_10=fe#2,<z_40_0=fe#2); for (i = 1;i < 10;++i) fe25519_square(>z_50_10=fe#2,>z_50_10=fe#2); */
  /* asm 2: fe25519_square(>z_50_10=&t1,<z_40_0=&t1); for (i = 1;i < 10;++i) fe25519_square(>z_50_10=&t1,>z_50_10=&t1); */
  fe25519_square(&t1,&t1); for (i = 1;i < 10;++i) fe25519_square(&t1,&t1);
  
  /* qhasm: z_50_0 = z_50_10*z_10_0 */
  /* asm 1: fe25519_mul(>z_50_0=fe#1,<z_50_10=fe#2,<z_10_0=fe#1); */
  /* asm 2: fe25519_mul(>z_50_0=&t0,<z_50_10=&t1,<z_10_0=&t0); */
  fe25519_mul(&t0,&t1,&t0);
  
  /* qhasm: z_100_50 = z_50_0^2^50 */
  /* asm 1: fe25519_square(>z_100_50=fe#2,<z_50_0=fe#1); for (i = 1;i < 50;++i) fe25519_square(>z_100_50=fe#2,>z_100_50=fe#2); */
  /* asm 2: fe25519_square(>z_100_50=&t1,<z_50_0=&t0); for (i = 1;i < 50;++i) fe25519_square(>z_100_50=&t1,>z_100_50=&t1); */
  fe25519_square(&t1,&t0); for (i = 1;i < 50;++i) fe25519_square(&t1,&t1);
  
  /* qhasm: z_100_0 = z_100_50*z_50_0 */
  /* asm 1: fe25519_mul(>z_100_0=fe#2,<z_100_50=fe#2,<z_50_0=fe#1); */
  /* asm 2: fe25519_mul(>z_100_0=&t1,<z_100_50=&t1,<z_50_0=&t0); */
  fe25519_mul(&t1,&t1,&t0);
  
  /* qhasm: z_200_100 = z_100_0^2^100 */
  /* asm 1: fe25519_square(>z_200_100=fe#3,<z_100_0=fe#2); for (i = 1;i < 100;++i) fe25519_square(>z_200_100=fe#3,>z_200_100=fe#3); */
  /* asm 2: fe25519_square(>z_200_100=&t2,<z_100_0=&t1); for (i = 1;i < 100;++i) fe25519_square(>z_200_100=&t2,>z_200_100=&t2); */
  fe25519_square(&t2,&t1); for (i = 1;i < 100;++i) fe25519_square(&t2,&t2);
  
  /* qhasm: z_200_0 = z_200_100*z_100_0 */
  /* asm 1: fe25519_mul(>z_200_0=fe#2,<z_200_100=fe#3,<z_100_0=fe#2); */
  /* asm 2: fe25519_mul(>z_200_0=&t1,<z_200_100=&t2,<z_100_0=&t1); */
  fe25519_mul(&t1,&t2,&t1);
  
  /* qhasm: z_250_50 = z_200_0^2^50 */
  /* asm 1: fe25519_square(>z_250_50=fe#2,<z_200_0=fe#2); for (i = 1;i < 50;++i) fe25519_square(>z_250_50=fe#2,>z_250_50=fe#2); */
  /* asm 2: fe25519_square(>z_250_50=&t1,<z_200_0=&t1); for (i = 1;i < 50;++i) fe25519_square(>z_250_50=&t1,>z_250_50=&t1); */
  fe25519_square(&t1,&t1); for (i = 1;i < 50;++i) fe25519_square(&t1,&t1);
  
  /* qhasm: z_250_0 = z_250_50*z_50_0 */
  /* asm 1: fe25519_mul(>z_250_0=fe#1,<z_250_50=fe#2,<z_50_0=fe#1); */
  /* asm 2: fe25519_mul(>z_250_0=&t0,<z_250_50=&t1,<z_50_0=&t0); */
  fe25519_mul(&t0,&t1,&t0);
  
  /* qhasm: z_252_2 = z_250_0^2^2 */
  /* asm 1: fe25519_square(>z_252_2=fe#1,<z_250_0=fe#1); for (i = 1;i < 2;++i) fe25519_square(>z_252_2=fe#1,>z_252_2=fe#1); */
  /* asm 2: fe25519_square(>z_252_2=&t0,<z_250_0=&t0); for (i = 1;i < 2;++i) fe25519_square(>z_252_2=&t0,>z_252_2=&t0); */
  fe25519_square(&t0,&t0); for (i = 1;i < 2;++i) fe25519_square(&t0,&t0);
  
  /* qhasm: z_252_3 = z_252_2*z1 */
  /* asm 1: fe25519_mul(>z_252_3=fe#12,<z_252_2=fe#1,<z1=fe#11); */
  /* asm 2: fe25519_mul(>z_252_3=out,<z_252_2=&t0,<z1=z); */
  fe25519_mul(out,&t0,z);
  
  /* qhasm: return */
  
  
  return;
}

/*
h = 2 * f * f
Can overlap h with f.

Preconditions:
   |f| bounded by 1.65*2^26,1.65*2^25,1.65*2^26,1.65*2^25,etc.

Postconditions:
   |h| bounded by 1.01*2^25,1.01*2^24,1.01*2^25,1.01*2^24,etc.
*/

/*
See fe_mul.c for discussion of implementation strategy.
*/
void fe25519_square_double(fe25519 *h,const fe25519 *f)
{
  crypto_int32 f0 = f->v[0];
  crypto_int32 f1 = f->v[1];
  crypto_int32 f2 = f->v[2];
  crypto_int32 f3 = f->v[3];
  crypto_int32 f4 = f->v[4];
  crypto_int32 f5 = f->v[5];
  crypto_int32 f6 = f->v[6];
  crypto_int32 f7 = f->v[7];
  crypto_int32 f8 = f->v[8];
  crypto_int32 f9 = f->v[9];
  crypto_int32 f0_2 = 2 * f0;
  crypto_int32 f1_2 = 2 * f1;
  crypto_int32 f2_2 = 2 * f2;
  crypto_int32 f3_2 = 2 * f3;
  crypto_int32 f4_2 = 2 * f4;
  crypto_int32 f5_2 = 2 * f5;
  crypto_int32 f6_2 = 2 * f6;
  crypto_int32 f7_2 = 2 * f7;
  crypto_int32 f5_38 = 38 * f5; /* 1.959375*2^30 */
  crypto_int32 f6_19 = 19 * f6; /* 1.959375*2^30 */
  crypto_int32 f7_38 = 38 * f7; /* 1.959375*2^30 */
  crypto_int32 f8_19 = 19 * f8; /* 1.959375*2^30 */
  crypto_int32 f9_38 = 38 * f9; /* 1.959375*2^30 */
  crypto_int64 f0f0    = f0   * (crypto_int64) f0;
  crypto_int64 f0f1_2  = f0_2 * (crypto_int64) f1;
  crypto_int64 f0f2_2  = f0_2 * (crypto_int64) f2;
  crypto_int64 f0f3_2  = f0_2 * (crypto_int64) f3;
  crypto_int64 f0f4_2  = f0_2 * (crypto_int64) f4;
  crypto_int64 f0f5_2  = f0_2 * (crypto_int64) f5;
  crypto_int64 f0f6_2  = f0_2 * (crypto_int64) f6;
  crypto_int64 f0f7_2  = f0_2 * (crypto_int64) f7;
  crypto_int64 f0f8_2  = f0_2 * (crypto_int64) f8;
  crypto_int64 f0f9_2  = f0_2 * (crypto_int64) f9;
  crypto_int64 f1f1_2  = f1_2 * (crypto_int64) f1;
  crypto_int64 f1f2_2  = f1_2 * (crypto_int64) f2;
  crypto_int64 f1f3_4  = f1_2 * (crypto_int64) f3_2;
  crypto_int64 f1f4_2  = f1_2 * (crypto_int64) f4;
  crypto_int64 f1f5_4  = f1_2 * (crypto_int64) f5_2;
  crypto_int64 f1f6_2  = f1_2 * (crypto_int64) f6;
  crypto_int64 f1f7_4  = f1_2 * (crypto_int64) f7_2;
  crypto_int64 f1f8_2  = f1_2 * (crypto_int64) f8;
  crypto_int64 f1f9_76 = f1_2 * (crypto_int64) f9_38;
  crypto_int64 f2f2    = f2   * (crypto_int64) f2;
  crypto_int64 f2f3_2  = f2_2 * (crypto_int64) f3;
  crypto_int64 f2f4_2  = f2_2 * (crypto_int64) f4;
  crypto_int64 f2f5_2  = f2_2 * (crypto_int64) f5;
  crypto_int64 f2f6_2  = f2_2 * (crypto_int64) f6;
  crypto_int64 f2f7_2  = f2_2 * (crypto_int64) f7;
  crypto_int64 f2f8_38 = f2_2 * (crypto_int64) f8_19;
  crypto_int64 f2f9_38 = f2   * (crypto_int64) f9_38;
  crypto_int64 f3f3_2  = f3_2 * (crypto_int64) f3;
  crypto_int64 f3f4_2  = f3_2 * (crypto_int64) f4;
  crypto_int64 f3f5_4  = f3_2 * (crypto_int64) f5_2;
  crypto_int64 f3f6_2  = f3_2 * (crypto_int64) f6;
  crypto_int64 f3f7_76 = f3_2 * (crypto_int64) f7_38;
  crypto_int64 f3f8_38 = f3_2 * (crypto_int64) f8_19;
  crypto_int64 f3f9_76 = f3_2 * (crypto_int64) f9_38;
  crypto_int64 f4f4    = f4   * (crypto_int64) f4;
  crypto_int64 f4f5_2  = f4_2 * (crypto_int64) f5;
  crypto_int64 f4f6_38 = f4_2 * (crypto_int64) f6_19;
  crypto_int64 f4f7_38 = f4   * (crypto_int64) f7_38;
  crypto_int64 f4f8_38 = f4_2 * (crypto_int64) f8_19;
  crypto_int64 f4f9_38 = f4   * (crypto_int64) f9_38;
  crypto_int64 f5f5_38 = f5   * (crypto_int64) f5_38;
  crypto_int64 f5f6_38 = f5_2 * (crypto_int64) f6_19;
  crypto_int64 f5f7_76 = f5_2 * (crypto_int64) f7_38;
  crypto_int64 f5f8_38 = f5_2 * (crypto_int64) f8_19;
  crypto_int64 f5f9_76 = f5_2 * (crypto_int64) f9_38;
  crypto_int64 f6f6_19 = f6   * (crypto_int64) f6_19;
  crypto_int64 f6f7_38 = f6   * (crypto_int64) f7_38;
  crypto_int64 f6f8_38 = f6_2 * (crypto_int64) f8_19;
  crypto_int64 f6f9_38 = f6   * (crypto_int64) f9_38;
  crypto_int64 f7f7_38 = f7   * (crypto_int64) f7_38;
  crypto_int64 f7f8_38 = f7_2 * (crypto_int64) f8_19;
  crypto_int64 f7f9_76 = f7_2 * (crypto_int64) f9_38;
  crypto_int64 f8f8_19 = f8   * (crypto_int64) f8_19;
  crypto_int64 f8f9_38 = f8   * (crypto_int64) f9_38;
  crypto_int64 f9f9_38 = f9   * (crypto_int64) f9_38;
  crypto_int64 h0 = f0f0  +f1f9_76+f2f8_38+f3f7_76+f4f6_38+f5f5_38;
  crypto_int64 h1 = f0f1_2+f2f9_38+f3f8_38+f4f7_38+f5f6_38;
  crypto_int64 h2 = f0f2_2+f1f1_2 +f3f9_76+f4f8_38+f5f7_76+f6f6_19;
  crypto_int64 h3 = f0f3_2+f1f2_2 +f4f9_38+f5f8_38+f6f7_38;
  crypto_int64 h4 = f0f4_2+f1f3_4 +f2f2   +f5f9_76+f6f8_38+f7f7_38;
  crypto_int64 h5 = f0f5_2+f1f4_2 +f2f3_2 +f6f9_38+f7f8_38;
  crypto_int64 h6 = f0f6_2+f1f5_4 +f2f4_2 +f3f3_2 +f7f9_76+f8f8_19;
  crypto_int64 h7 = f0f7_2+f1f6_2 +f2f5_2 +f3f4_2 +f8f9_38;
  crypto_int64 h8 = f0f8_2+f1f7_4 +f2f6_2 +f3f5_4 +f4f4   +f9f9_38;
  crypto_int64 h9 = f0f9_2+f1f8_2 +f2f7_2 +f3f6_2 +f4f5_2;
  crypto_int64 carry0;
  crypto_int64 carry1;
  crypto_int64 carry2;
  crypto_int64 carry3;
  crypto_int64 carry4;
  crypto_int64 carry5;
  crypto_int64 carry6;
  crypto_int64 carry7;
  crypto_int64 carry8;
  crypto_int64 carry9;

  h0 += h0;
  h1 += h1;
  h2 += h2;
  h3 += h3;
  h4 += h4;
  h5 += h5;
  h6 += h6;
  h7 += h7;
  h8 += h8;
  h9 += h9;

  carry0 = (h0 + (crypto_int64) (1<<25)) >> 26; h1 += carry0; h0 -= carry0 << 26;
  carry4 = (h4 + (crypto_int64) (1<<25)) >> 26; h5 += carry4; h4 -= carry4 << 26;

  carry1 = (h1 + (crypto_int64) (1<<24)) >> 25; h2 += carry1; h1 -= carry1 << 25;
  carry5 = (h5 + (crypto_int64) (1<<24)) >> 25; h6 += carry5; h5 -= carry5 << 25;

  carry2 = (h2 + (crypto_int64) (1<<25)) >> 26; h3 += carry2; h2 -= carry2 << 26;
  carry6 = (h6 + (crypto_int64) (1<<25)) >> 26; h7 += carry6; h6 -= carry6 << 26;

  carry3 = (h3 + (crypto_int64) (1<<24)) >> 25; h4 += carry3; h3 -= carry3 << 25;
  carry7 = (h7 + (crypto_int64) (1<<24)) >> 25; h8 += carry7; h7 -= carry7 << 25;

  carry4 = (h4 + (crypto_int64) (1<<25)) >> 26; h5 += carry4; h4 -= carry4 << 26;
  carry8 = (h8 + (crypto_int64) (1<<25)) >> 26; h9 += carry8; h8 -= carry8 << 26;

  carry9 = (h9 + (crypto_int64) (1<<24)) >> 25; h0 += carry9 * 19; h9 -= carry9 << 25;

  carry0 = (h0 + (crypto_int64) (1<<25)) >> 26; h1 += carry0; h0 -= carry0 << 26;

  h->v[0] = h0;
  h->v[1] = h1;
  h->v[2] = h2;
  h->v[3] = h3;
  h->v[4] = h4;
  h->v[5] = h5;
  h->v[6] = h6;
  h->v[7] = h7;
  h->v[8] = h8;
  h->v[9] = h9;
}

void fe25519_sqrt(fe25519 *r, const fe25519 *x)
{
  fe25519 t;
  fe25519_invsqrt(&t, x);
  fe25519_mul(r, &t, x);
}

void fe25519_invsqrt(fe25519 *r, const fe25519 *x)
{
  fe25519 den2, den3, den4, den6, chk, t;
  fe25519_square(&den2, x);
  fe25519_mul(&den3, &den2, x);
  
  fe25519_square(&den4, &den2);
  fe25519_mul(&den6, &den2, &den4);
  fe25519_mul(&t, &den6, x); // r is now x^7
  
  fe25519_pow2523(&t, &t);
  fe25519_mul(&t, &t, &den3);
  
  fe25519_square(&chk, &t);
  fe25519_mul(&chk, &chk, x);
  
  if(!fe25519_isone(&chk)) //XXX: Make constant time
    fe25519_mul(&t, &t, &fe25519_sqrtm1);
  
  *r = t;
}






// -- group.c --

/* 
 * Arithmetic on the twisted Edwards curve -x^2 + y^2 = 1 + dx^2y^2 
 * with d = -(121665/121666) = 37095705934669439343138083508754565189542113879843219016388785533085940283555
 * Base point: (15112221349535400772501151409588531511454012693041857206046113283949847762202,46316835694926478169428394003475163141307993866256225615783033603165251855960);
 */


static const fe25519 ge25519_ecd = {{-10913610, 13857413, -15372611, 6949391, 114729, -8787816, -6275908, -3247719, -18696448, -12055116}};
static const fe25519 ge25519_ec2d = {{-21827239, -5839606, -30745221, 13898782, 229458, 15978800, -12551817, -6495438, 29715968, 9444199}};
static const fe25519 ge25519_magic = {{-6111485, -4156064, 27798727, -12243468, 25904040, -120897, -20826367, 7060776, -6093568, 1986012}};
const group_ge group_ge_neutral = {{{0, 0, 0, 0, 0, 0, 0, 0, 0, 0}},
                                    {{1, 0, 0, 0, 0, 0, 0, 0, 0, 0}},
                                    {{1, 0, 0, 0, 0, 0, 0, 0, 0, 0}},
                                    {{0, 0, 0, 0, 0, 0, 0, 0, 0, 0}}};

#define ge25519_p3 group_ge

typedef struct
{
  fe25519 x;
  fe25519 z;
  fe25519 y;
  fe25519 t;
} ge25519_p1p1;

typedef struct
{
  fe25519 x;
  fe25519 y;
  fe25519 z;
} ge25519_p2;

typedef struct
{
  fe25519 x;
  fe25519 y;
} ge25519_aff;


/* Multiples of the base point in affine representation */
static const ge25519_aff ge25519_base_multiples_affine[425] = {
#include "base_multiples.data"
};


static void ge25519_mixadd2(ge25519_p3 *r, const ge25519_aff *q)
{
  fe25519 a,b,t1,t2,c,d,e,f,g,h,qt;
  fe25519_mul(&qt, &q->x, &q->y);
  fe25519_sub(&a, &r->y, &r->x); /* A = (Y1-X1)*(Y2-X2) */
  fe25519_add(&b, &r->y, &r->x); /* B = (Y1+X1)*(Y2+X2) */
  fe25519_sub(&t1, &q->y, &q->x);
  fe25519_add(&t2, &q->y, &q->x);
  fe25519_mul(&a, &a, &t1);
  fe25519_mul(&b, &b, &t2);
  fe25519_sub(&e, &b, &a); /* E = B-A */
  fe25519_add(&h, &b, &a); /* H = B+A */
  fe25519_mul(&c, &r->t, &qt); /* C = T1*k*T2 */
  fe25519_mul(&c, &c, &ge25519_ec2d);
  fe25519_add(&d, &r->z, &r->z); /* D = Z1*2 */
  fe25519_sub(&f, &d, &c); /* F = D-C */
  fe25519_add(&g, &d, &c); /* G = D+C */
  fe25519_mul(&r->x, &e, &f);
  fe25519_mul(&r->y, &h, &g);
  fe25519_mul(&r->z, &g, &f);
  fe25519_mul(&r->t, &e, &h);
}

static void p1p1_to_p2(ge25519_p2 *r, const ge25519_p1p1 *p)
{
  fe25519_mul(&r->x, &p->x, &p->t);
  fe25519_mul(&r->y, &p->y, &p->z);
  fe25519_mul(&r->z, &p->z, &p->t);
}

static void p1p1_to_p3(ge25519_p3 *r, const ge25519_p1p1 *p)
{
  p1p1_to_p2((ge25519_p2 *)r, p);
  fe25519_mul(&r->t, &p->x, &p->y);
}

static void add_p1p1(ge25519_p1p1 *r, const ge25519_p3 *p, const ge25519_p3 *q)
{
  fe25519 a, b, c, d, t;
  
  fe25519_sub(&a, &p->y, &p->x); /* A = (Y1-X1)*(Y2-X2) */
  fe25519_sub(&t, &q->y, &q->x);
  fe25519_mul(&a, &a, &t);
  fe25519_add(&b, &p->x, &p->y); /* B = (Y1+X1)*(Y2+X2) */
  fe25519_add(&t, &q->x, &q->y);
  fe25519_mul(&b, &b, &t);
  fe25519_mul(&c, &p->t, &q->t); /* C = T1*k*T2 */
  fe25519_mul(&c, &c, &ge25519_ec2d);
  fe25519_mul(&d, &p->z, &q->z); /* D = Z1*2*Z2 */
  fe25519_add(&d, &d, &d);
  fe25519_sub(&r->x, &b, &a); /* E = B-A */
  fe25519_sub(&r->t, &d, &c); /* F = D-C */
  fe25519_add(&r->z, &d, &c); /* G = D+C */
  fe25519_add(&r->y, &b, &a); /* H = B+A */
}

/* See http://www.hyperelliptic.org/EFD/g1p/auto-twisted-extended-1.html#doubling-dbl-2008-hwcd */
static void dbl_p1p1(ge25519_p1p1 *r, const ge25519_p2 *p)
{
  fe25519 a,b,c,d;
  fe25519_square(&a, &p->x);
  fe25519_square(&b, &p->y);
  fe25519_square_double(&c, &p->z);
  fe25519_neg(&d, &a);

  fe25519_add(&r->x, &p->x, &p->y);
  fe25519_square(&r->x, &r->x);
  fe25519_sub(&r->x, &r->x, &a);
  fe25519_sub(&r->x, &r->x, &b);
  fe25519_add(&r->z, &d, &b);
  fe25519_sub(&r->t, &r->z, &c);
  fe25519_sub(&r->y, &d, &b);
}

/* Constant-time version of: if(b) r = p */
static void cmov_aff(ge25519_aff *r, const ge25519_aff *p, unsigned char b)
{
  fe25519_cmov(&r->x, &p->x, b);
  fe25519_cmov(&r->y, &p->y, b);
}

static unsigned char group_c_static_equal(signed char b,signed char c)
{
  unsigned char ub = b;
  unsigned char uc = c;
  unsigned char x = ub ^ uc; /* 0: yes; 1..255: no */
  crypto_uint32 y = x; /* 0: yes; 1..255: no */
  y -= 1; /* 4294967295: yes; 0..254: no */
  y >>= 31; /* 1: yes; 0: no */
  return y;
}

static unsigned char negative(signed char b)
{
  unsigned long long x = b; /* 18446744073709551361..18446744073709551615: yes; 0..255: no */
  x >>= 63; /* 1: yes; 0: no */
  return x;
}

static void choose_t_aff(ge25519_aff *t, unsigned long long pos, signed char b)
{
  fe25519 v;
  *t = ge25519_base_multiples_affine[5*pos+0];
  cmov_aff(t, &ge25519_base_multiples_affine[5*pos+1],group_c_static_equal(b,1) | group_c_static_equal(b,-1));
  cmov_aff(t, &ge25519_base_multiples_affine[5*pos+2],group_c_static_equal(b,2) | group_c_static_equal(b,-2));
  cmov_aff(t, &ge25519_base_multiples_affine[5*pos+3],group_c_static_equal(b,3) | group_c_static_equal(b,-3));
  cmov_aff(t, &ge25519_base_multiples_affine[5*pos+4],group_c_static_equal(b,-4));
  fe25519_neg(&v, &t->x);
  fe25519_cmov(&t->x, &v, negative(b));
}


static void choose_t(group_ge *t, const group_ge *pre, signed char b)
{
  fe25519 v;
  signed char j;
  unsigned char c;

  *t = pre[0];
  for(j=1;j<=16;j++)
  {
    c = group_c_static_equal(b,j) | group_c_static_equal(-b,j);
    fe25519_cmov(&t->x, &pre[j].x,c);
    fe25519_cmov(&t->y, &pre[j].y,c);
    fe25519_cmov(&t->z, &pre[j].z,c);
    fe25519_cmov(&t->t, &pre[j].t,c);
  }
  fe25519_neg(&v, &t->x);
  fe25519_cmov(&t->x, &v, negative(b));
  fe25519_neg(&v, &t->t);
  fe25519_cmov(&t->t, &v, negative(b));
}


// ==================================================================================
//                                    API FUNCTIONS
// ==================================================================================

/*
const group_ge group_ge_base = {{{133, 0, 0, 37120, 137, 0, 0, 42983, 58, 0, 7808, 5998, 12, 49152, 49039, 1015}},
                               {{65422, 65535, 65535, 5631, 65418, 65535, 65535, 47417, 65485, 65535, 12031, 41670, 65525, 32767, 42226, 8491}},
                               {{65422, 65535, 65535, 5631, 65418, 65535, 65535, 47417, 65485, 65535, 12031, 41670, 65525, 32767, 42226, 8491}}};
                               */

const group_ge group_ge_base = {{{-14297830, -7645148, 16144683, -16471763, 27570974, -2696100, -26142465, 8378389, 20764389, 8758491}},
                                {{-26843541, -6710886, 13421773, -13421773, 26843546, 6710886, -13421773, 13421773, -26843546, -6710886}},
                                {{1, 0, 0, 0, 0, 0, 0, 0, 0, 0}},
                                {{28827062, -6116119, -27349572, 244363, 8635006, 11264893, 19351346, 13413597, 16611511, -6414980}}};

int group_ge_unpack(group_ge *r, const unsigned char x[GROUP_GE_PACKEDBYTES])
{
  fe25519 s, s2, chk, yden, ynum, yden2, xden2, isr, xdeninv, ydeninv, t;
  int ret;
  unsigned char b;

  fe25519_unpack(&s, x);

  /* s = cls.bytesToGf(s,mustBePositive=True) */
  ret = fe25519_isnegative(&s);

  /* yden     = 1-a*s^2    // 1+s^2 */
  /* ynum     = 1+a*s^2    // 1-s^2 */
  fe25519_square(&s2, &s);
  fe25519_add(&yden,&fe25519_one,&s2);
  fe25519_sub(&ynum,&fe25519_one,&s2);
  
  /* yden_sqr = yden^2 */
  /* xden_sqr = a*d*ynum^2 - yden_sqr */
  fe25519_square(&yden2, &yden);
  fe25519_square(&xden2, &ynum);
  fe25519_mul(&xden2, &xden2, &ge25519_ecd); // d*ynum^2
  fe25519_add(&xden2, &xden2, &yden2); // d*ynum2+yden2
  fe25519_neg(&xden2, &xden2); // -d*ynum2-yden2
  
  /* isr = isqrt(xden_sqr * yden_sqr) */
  fe25519_mul(&t, &xden2, &yden2);
  fe25519_invsqrt(&isr, &t);

  //Check inverse square root!
  fe25519_square(&chk, &isr);
  fe25519_mul(&chk, &chk, &t);

  ret |= !fe25519_isone(&chk);

  /* xden_inv = isr * yden */
  fe25519_mul(&xdeninv, &isr, &yden);
  
        
  /* yden_inv = xden_inv * isr * xden_sqr */
  fe25519_mul(&ydeninv, &xdeninv, &isr);
  fe25519_mul(&ydeninv, &ydeninv, &xden2);

  /* x = 2*s*xden_inv */
  fe25519_mul(&r->x, &s, &xdeninv);
  fe25519_double(&r->x, &r->x);

  /* if negative(x): x = -x */
  b = fe25519_isnegative(&r->x);
  fe25519_neg(&t, &r->x);
  fe25519_cmov(&r->x, &t, b);

        
  /* y = ynum * yden_inv */
  fe25519_mul(&r->y, &ynum, &ydeninv);

  r->z = fe25519_one;

  /* if cls.cofactor==8 and (negative(x*y) or y==0):
       raise InvalidEncodingException("x*y is invalid: %d, %d" % (x,y)) */
  fe25519_mul(&r->t, &r->x, &r->y);
  ret |= fe25519_isnegative(&r->t);
  ret |= fe25519_iszero(&r->y);


  // Zero all coordinates of point for invalid input; produce invalid point
  fe25519_cmov(&r->x, &fe25519_zero, ret);
  fe25519_cmov(&r->y, &fe25519_zero, ret);
  fe25519_cmov(&r->z, &fe25519_zero, ret);
  fe25519_cmov(&r->t, &fe25519_zero, ret);

  return -ret;
}

// Return x if x is positive, else return -x.
void fe25519_abs(fe25519* x, const fe25519* y)
{
    fe25519 negY;
    *x = *y;
    fe25519_neg(&negY, y);
    fe25519_cmov(x, &negY, fe25519_isnegative(x));
}

// Sets r to sqrt(x) or sqrt(i * x).  Returns 1 if x is a square.
int fe25519_sqrti(fe25519 *r, const fe25519 *x)
{
  int b;
  fe25519 t, corr;
  b = fe25519_invsqrti(&t, x);
  fe25519_setone(&corr);
  fe25519_cmov(&corr, &fe25519_sqrtm1, 1 - b);
  fe25519_mul(&t, &t, &corr);
  fe25519_mul(r, &t, x);
  return b;
}

// Sets r to 1/sqrt(x) or 1/sqrt(i*x).  Returns whether x was a square.
int fe25519_invsqrti(fe25519 *r, const fe25519 *x)
{
  int inCaseA, inCaseB, inCaseD;
  fe25519 den2, den3, den4, den6, chk, t, corr;
  fe25519_square(&den2, x);
  fe25519_mul(&den3, &den2, x);
  
  fe25519_square(&den4, &den2);
  fe25519_mul(&den6, &den2, &den4);
  fe25519_mul(&t, &den6, x); // r is now x^7
  
  fe25519_pow2523(&t, &t);
  fe25519_mul(&t, &t, &den3);
   
  // case       A           B            C             D
  // ---------------------------------------------------------------
  // t          1/sqrt(x)   -i/sqrt(x)   1/sqrt(i*x)   -i/sqrt(i*x)
  // chk        1           -1           -i            i
  // corr       1           i            1             i
  // ret        1           1            0             0
  fe25519_square(&chk, &t);
  fe25519_mul(&chk, &chk, x);

  inCaseA = fe25519_isone(&chk);
  inCaseD = fe25519_iseq(&chk, &fe25519_sqrtm1);
  fe25519_neg(&chk, &chk);
  inCaseB = fe25519_isone(&chk);

  fe25519_setone(&corr);
  fe25519_cmov(&corr, &fe25519_sqrtm1, inCaseB + inCaseD);
  fe25519_mul(&t, &t, &corr);
  
  *r = t;

  return inCaseA + inCaseB;
}


void group_ge_pack(unsigned char r[GROUP_GE_PACKEDBYTES], const group_ge *x)
{
  fe25519 d, u1, u2, isr, i1, i2, zinv, deninv, nx, ny, s;
  unsigned char b;

  /* u1    = mneg*(z+y)*(z-y) */
  fe25519_add(&d, &x->z, &x->y);
  fe25519_sub(&u1, &x->z, &x->y);
  fe25519_mul(&u1, &u1, &d);

  /* u2    = x*y # = t*z */
  fe25519_mul(&u2, &x->x, &x->y);

  /* isr   = isqrt(u1*u2^2) */
  fe25519_square(&isr, &u2);
  fe25519_mul(&isr, &isr, &u1);
  fe25519_invsqrt(&isr, &isr);

  /* i1    = isr*u1 # sqrt(mneg*(z+y)*(z-y))/(x*y) */
  fe25519_mul(&i1, &isr, &u1);
  
  /* i2    = isr*u2 # 1/sqrt(a*(y+z)*(y-z)) */
  fe25519_mul(&i2, &isr, &u2);

  /* z_inv = i1*i2*t # 1/z */
  fe25519_mul(&zinv, &i1, &i2);
  fe25519_mul(&zinv, &zinv, &x->t);

  /* if negative(t*z_inv):
       x,y = y*self.i,x*self.i
       den_inv = self.magic * i1 */
  fe25519_mul(&d, &zinv, &x->t);
  b = !fe25519_isnegative(&d);

  fe25519_mul(&nx, &x->y, &fe25519_sqrtm1);
  fe25519_mul(&ny, &x->x, &fe25519_sqrtm1);
  fe25519_mul(&deninv, &ge25519_magic, &i1);

  fe25519_cmov(&nx, &x->x, b);
  fe25519_cmov(&ny, &x->y, b);
  fe25519_cmov(&deninv, &i2, b);

  /* if negative(x*z_inv): y = -y */
  fe25519_mul(&d, &nx, &zinv);
  b = fe25519_isnegative(&d);
  fe25519_neg(&d, &ny);
  fe25519_cmov(&ny, &d, b);

  /* s = (z-y) * den_inv */
  fe25519_sub(&s, &x->z, &ny);
  fe25519_mul(&s, &s, &deninv);

  /* return self.gfToBytes(s,mustBePositive=True) */
  b = fe25519_isnegative(&s);
  fe25519_neg(&d, &s);
  fe25519_cmov(&s, &d, b);

  fe25519_pack(r, &s);
}

void group_ge_add(group_ge *r, const group_ge *x, const group_ge *y)
{
  ge25519_p1p1 t;
  add_p1p1(&t, x, y);
  p1p1_to_p3(r,&t);
}

void group_ge_double(group_ge *r, const group_ge *x)
{
  ge25519_p1p1 t;
  dbl_p1p1(&t, (ge25519_p2 *)x);
  p1p1_to_p3(r,&t);
}

void group_ge_negate(group_ge *r, const group_ge *x)
{
  fe25519_neg(&r->x, &x->x);
  r->y = x->y;
  r->z = x->z;
  fe25519_neg(&r->t, &x->t);
}

void group_ge_scalarmult(group_ge *r, const group_ge *x, const group_scalar *s)
{
  group_ge precomp[17],t;
  int i, j;
  signed char win5[51];

  scalar_window5(win5, s);

  //precomputation:
  precomp[0] = group_ge_neutral;
  precomp[1] = *x;
  for (i = 2; i < 16; i+=2)
  {
    group_ge_double(precomp+i,precomp+i/2);
    group_ge_add(precomp+i+1,precomp+i,precomp+1);
  }
  group_ge_double(precomp+16,precomp+8);
  
  *r = group_ge_neutral;
	for (i = 50; i >= 0; i--)
	{
		for (j = 0; j < 5; j++)
			group_ge_double(r, r); //change to not compute t all the time
    choose_t(&t, precomp, win5[i]);

		group_ge_add(r, r, &t);
  }
}

void group_ge_scalarmult_base(group_ge *r, const group_scalar *s)
{
  signed char b[85];
  int i;
  ge25519_aff t;
  scalar_window3(b,s);

  choose_t_aff((ge25519_aff *)r, 0, b[0]);
  r->z = fe25519_one;
  fe25519_mul(&r->t, &r->x, &r->y);
  for(i=1;i<85;i++)
  {
    choose_t_aff(&t, (unsigned long long) i, b[i]);
    ge25519_mixadd2(r, &t);
  }
}

void group_ge_multiscalarmult(group_ge *r, const group_ge *x, const group_scalar *s, unsigned long long xlen)
{
  //XXX: Use Strauss 
  unsigned long long i;
  group_ge t;
  *r = group_ge_neutral;
  for(i=0;i<xlen;i++)
  {
    group_ge_scalarmult(&t,x+i,s+i);
    group_ge_add(r,r,&t);
  }
}

int  group_ge_equals(const group_ge *x, const group_ge *y)
{
  fe25519 x1y2, x2y1, x1x2, y1y2;
  int r;
  
  fe25519_mul(&x1y2, &x->x, &y->y);
  fe25519_mul(&x2y1, &y->x, &x->y);

  r =  fe25519_iseq(&x1y2, &x2y1);

  fe25519_mul(&x1x2, &x->x, &y->x);
  fe25519_mul(&y1y2, &x->y, &y->y);
  
  r |=  fe25519_iseq(&x1x2, &y1y2);

  return r;
}

int  group_ge_isneutral(const group_ge *x)
{
  int r;
  group_ge t;

  // double three times for decaf8
  group_ge_double(&t, x);
  group_ge_double(&t, &t);
  group_ge_double(&t, &t);

  r = 1-fe25519_iszero(&t.x);
  r |= 1-fe25519_iseq(&t.y, &t.z);
  return 1-r;
}




void group_ge_add_publicinputs(group_ge *r, const group_ge *x, const group_ge *y)
{
  group_ge_add(r,x,y);
}

void group_ge_double_publicinputs(group_ge *r, const group_ge *x)
{
  group_ge_double(r,x);
}

void group_ge_negate_publicinputs(group_ge *r, const group_ge *x)
{
  group_ge_negate(r,x);
}

void group_ge_scalarmult_publicinputs(group_ge *r, const group_ge *x, const group_scalar *s)
{
  //XXX: Use sliding window
  group_ge_scalarmult(r, x, s);
}


void group_ge_scalarmult_base_publicinputs(group_ge *r, const group_scalar *s)
{
  //group_ge_scalarmult_publicinputs(r,&group_ge_base,s);
  group_ge_scalarmult_base(r,s);
}

void group_ge_multiscalarmult_publicinputs(group_ge *r, const group_ge *x, const group_scalar *s, unsigned long long xlen)
{
  //XXX: Use Bos-Coster (and something else for small values of xlen)
  group_ge_multiscalarmult(r,x,s,xlen);
}

int  group_ge_equals_publicinputs(const group_ge *x, const group_ge *y)
{
  return group_ge_equals(x,y);
}

int  group_ge_isneutral_publicinputs(const group_ge *x)
{
  return group_ge_isneutral(x);
}

/*
void ge_print(const group_ge *a) {
 fe25519_print(&a->x);
 fe25519_print(&a->y);
 fe25519_print(&a->z);
 fe25519_print(&a->t);
}
*/

void group_ge_from_jacobi_quartic(group_ge *x, 
		const fe25519 *s, const fe25519 *t)
{
    ge25519_p1p1 res;
    fe25519 s2;

    fe25519_square(&s2, s);

    // Set x to 2 * s * 1/sqrt(-d-1)
    fe25519_double(&res.x, s);
    fe25519_mul(&res.x, &res.x, &ge25519_magic);

    // Set z to t
    res.z = *t;

    // Set y to 1-s^2
    fe25519_sub(&res.y, &fe25519_one, &s2);

    // Set t to 1+s^2
    fe25519_add(&res.t, &fe25519_one, &s2);
    p1p1_to_p3(x, &res);
}

// Compute the point corresponding to the scalar r0 in the
// Elligator2 encoding adapted to Ristretto.
void group_ge_elligator(group_ge *x, const fe25519 *r0)
{
    fe25519 r, rPlusD, rPlusOne, ecd2, D, N, ND, sqrt, twiddle, sgn;
    fe25519 s, t, dMinusOneSquared, rSubOne, r0i, sNeg;
    int b;

    // r := i * r0^2
    fe25519_mul(&r0i, r0, &fe25519_sqrtm1);
    fe25519_mul(&r, r0, &r0i);

    // D := -((d*r)+1) * (r + d)
    fe25519_add(&rPlusD, &ge25519_ecd, &r);
    fe25519_mul(&D, &ge25519_ecd, &r);
    fe25519_add(&D, &D, &fe25519_one);
    fe25519_mul(&D, &D, &rPlusD);
    fe25519_neg(&D, &D);

    // N := -(d^2 - 1)(r + 1)
    fe25519_square(&ecd2, &ge25519_ecd);
    fe25519_sub(&N, &ecd2, &fe25519_one);
    fe25519_neg(&N, &N); // TODO add -(d^2-1) as a constant
    fe25519_add(&rPlusOne, &r, &fe25519_one);
    fe25519_mul(&N, &N, &rPlusOne);

    // sqrt is the inverse square root of N*D or of i*N*D.  b=1 iff n1 is square.
    fe25519_mul(&ND, &N, &D);
    b = fe25519_invsqrti(&sqrt, &ND);
    fe25519_abs(&sqrt, &sqrt);

    fe25519_setone(&twiddle);
    fe25519_cmov(&twiddle, &r0i, 1 - b);
    fe25519_setone(&sgn);
    fe25519_cmov(&sgn, &fe25519_m1, 1 - b);
    fe25519_mul(&sqrt, &sqrt, &twiddle);

    // s = N * sqrt(N*D) * twiddle
    fe25519_mul(&s, &sqrt, &N);

    // t = -sgn * sqrt * s * (r-1) * (d-1)^2 - 1
    fe25519_neg(&t, &sgn);
    fe25519_mul(&t, &sqrt, &t);
    fe25519_mul(&t, &s, &t);
    fe25519_sub(&dMinusOneSquared, &ge25519_ecd, &fe25519_one);
    fe25519_square(&dMinusOneSquared, &dMinusOneSquared); // TODO make constant
    fe25519_mul(&t, &dMinusOneSquared, &t);
    fe25519_sub(&rSubOne, &r, &fe25519_one);
    fe25519_mul(&t, &rSubOne, &t);
    fe25519_sub(&t, &t, &fe25519_one);

    fe25519_neg(&sNeg, &s);
    fe25519_cmov(&s, &sNeg, fe25519_isnegative(&s) == b);

    group_ge_from_jacobi_quartic(x, &s, &t);
}
