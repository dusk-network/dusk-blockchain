#include <stdint.h>

// -- crypto_(u)int{32,64}.h --


typedef unsigned int crypto_uint32;
typedef int crypto_int32;
typedef int64_t crypto_int64;
typedef uint64_t crypto_uint64;

// -- fe25519.h --

typedef struct
{
  crypto_int32 v[10];
}
fe25519;

extern const fe25519 fe25519_zero;
extern const fe25519 fe25519_one;
extern const fe25519 fe25519_two;

extern const fe25519 fe25519_sqrtm1;
extern const fe25519 fe25519_msqrtm1;
extern const fe25519 fe25519_m1;

//void fe25519_freeze(fe25519 *r);

void fe25519_unpack(fe25519 *r, const unsigned char x[32]);

void fe25519_pack(unsigned char r[32], const fe25519 *x);

int fe25519_iszero(const fe25519 *x);

int fe25519_isone(const fe25519 *x);

int fe25519_isnegative(const fe25519 *x);

int fe25519_iseq(const fe25519 *x, const fe25519 *y);

int fe25519_iseq_vartime(const fe25519 *x, const fe25519 *y);

void fe25519_cmov(fe25519 *r, const fe25519 *x, unsigned char b);

void fe25519_setone(fe25519 *r);

void fe25519_setzero(fe25519 *r);

void fe25519_neg(fe25519 *r, const fe25519 *x);

unsigned char fe25519_getparity(const fe25519 *x);

void fe25519_add(fe25519 *r, const fe25519 *x, const fe25519 *y);

void fe25519_double(fe25519 *r, const fe25519 *x);
void fe25519_triple(fe25519 *r, const fe25519 *x);

void fe25519_sub(fe25519 *r, const fe25519 *x, const fe25519 *y);

void fe25519_mul(fe25519 *r, const fe25519 *x, const fe25519 *y);

void fe25519_square(fe25519 *r, const fe25519 *x);
void fe25519_square_double(fe25519 *h,const fe25519 *f);

void fe25519_invert(fe25519 *r, const fe25519 *x);

void fe25519_pow2523(fe25519 *r, const fe25519 *x);

void fe25519_invsqrt(fe25519 *r, const fe25519 *x);

int fe25519_invsqrti(fe25519 *r, const fe25519 *x);

int fe25519_sqrti(fe25519 *r, const fe25519 *x);

void fe25519_sqrt(fe25519 *r, const fe25519 *x);

//void fe25519_print(const fe25519 *x);


// -- scalar.h --
#define GROUP_SCALAR_PACKEDBYTES 32

typedef struct 
{
  uint32_t v[32]; 
}
group_scalar;

extern const group_scalar group_scalar_zero;
extern const group_scalar group_scalar_one;

int group_scalar_unpack(group_scalar *r, const unsigned char x[GROUP_SCALAR_PACKEDBYTES]);
void group_scalar_pack(unsigned char s[GROUP_SCALAR_PACKEDBYTES], const group_scalar *r);

void group_scalar_setzero(group_scalar *r);
void group_scalar_setone(group_scalar *r);
//void group_scalar_setrandom(group_scalar *r); // Removed to avoid dependency on platform specific randombytes
void group_scalar_add(group_scalar *r, const group_scalar *x, const group_scalar *y);
void group_scalar_sub(group_scalar *r, const group_scalar *x, const group_scalar *y);
void group_scalar_negate(group_scalar *r, const group_scalar *x);
void group_scalar_mul(group_scalar *r, const group_scalar *x, const group_scalar *y);
void group_scalar_square(group_scalar *r, const group_scalar *x);
void group_scalar_invert(group_scalar *r, const group_scalar *x);

int  group_scalar_isone(const group_scalar *x);
int  group_scalar_iszero(const group_scalar *x);
int  group_scalar_equals(const group_scalar *x,  const group_scalar *y);


// Additional functions, not required by API
int  scalar_tstbit(const group_scalar *x, const unsigned int pos);
int  scalar_bitlen(const group_scalar *x);
void scalar_window3(signed char r[85], const group_scalar *x);
void scalar_window5(signed char r[51], const group_scalar *s);
void scalar_slide(signed char r[256], const group_scalar *s, int swindowsize);

void scalar_from64bytes(group_scalar *r, const unsigned char h[64]);


// -- group.h --

#define GROUP_GE_PACKEDBYTES 32

typedef struct
{	
	fe25519 x;
	fe25519 y;
	fe25519 z;
	fe25519 t;
} group_ge;

extern const group_ge group_ge_base;
extern const group_ge group_ge_neutral;

// Constant-time versions
int  group_ge_unpack(group_ge *r, const unsigned char x[GROUP_GE_PACKEDBYTES]);
void group_ge_pack(unsigned char r[GROUP_GE_PACKEDBYTES], const group_ge *x);

void group_ge_add(group_ge *r, const group_ge *x, const group_ge *y);
void group_ge_double(group_ge *r, const group_ge *x);
void group_ge_negate(group_ge *r, const group_ge *x);
void group_ge_scalarmult(group_ge *r, const group_ge *x, const group_scalar *s);
void group_ge_scalarmult_base(group_ge *r, const group_scalar *s);
void group_ge_multiscalarmult(group_ge *r, const group_ge *x, const group_scalar *s, unsigned long long xlen);
int  group_ge_equals(const group_ge *x, const group_ge *y);
int  group_ge_isneutral(const group_ge *x);

// Non-constant-time versions
void group_ge_add_publicinputs(group_ge *r, const group_ge *x, const group_ge *y);
void group_ge_double_publicinputs(group_ge *r, const group_ge *x);
void group_ge_negate_publicinputs(group_ge *r, const group_ge *x);
void group_ge_scalarmult_publicinputs(group_ge *r, const group_ge *x, const group_scalar *s);
void group_ge_scalarmult_base_publicinputs(group_ge *r, const group_scalar *s);
void group_ge_multiscalarmult_publicinputs(group_ge *r, const group_ge *x, const group_scalar *s, unsigned long long xlen);
int  group_ge_equals_publicinputs(const group_ge *x, const group_ge *y);
int  group_ge_isneutral_publicinputs(const group_ge *x);

// Not required by API
//void ge_print(const group_ge *x);

void group_ge_from_jacobi_quartic(group_ge *x, 
		const fe25519 *s, const fe25519 *t);
void group_ge_elligator(group_ge *x, const fe25519 *r0);

