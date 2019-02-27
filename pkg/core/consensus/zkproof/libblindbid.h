/* libblindbid Header Version 0.1.0 */

#include <stdlib.h>
#include <stdbool.h>

struct Buffer {
  const uint8_t *ptr;
  size_t len;
};

void prog(const uint8_t *seed_ptr,
          const uint8_t *k_ptr,
          const uint8_t *d_ptr,
          uint8_t *q_ptr,
          uint8_t *x_ptr,
          uint8_t *y_ptr,
          uint8_t *y_inv_ptr,
          uint8_t *z_img_ptr);

struct Buffer *prove(const uint8_t *d_ptr,
            const uint8_t *k_ptr,
            const uint8_t *y_ptr,
            const uint8_t *y_inv_ptr,
            const uint8_t *q_ptr,
            const uint8_t *z_img_ptr,
            const uint8_t *seed_ptr,
            struct Buffer *pub_list,
            uintptr_t toggle);

bool verify(struct Buffer *buff,
            const uint8_t *seed_ptr,
            struct Buffer *pub_list,
            const uint8_t *q_ptr,
            const uint8_t *z_img_ptr);
