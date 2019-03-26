/* libblindbid Header Version 0.1.0 */

#include <stdlib.h>
#include <stdbool.h>

struct Buffer {
  const uint8_t *ptr;
  size_t len;
};


struct Buffer *prove(const uint8_t *d_ptr,
            const uint8_t *k_ptr,
            const uint8_t *y_ptr,
            const uint8_t *y_inv_ptr,
            const uint8_t *q_ptr,
            const uint8_t *z_img_ptr,
            const uint8_t *seed_ptr,
            struct Buffer *pub_list,
            struct Buffer *constants,
            uint8_t toggle);

bool verify(struct Buffer *buff,
            const uint8_t *seed_ptr,
            struct Buffer *pub_list,
            const uint8_t *q_ptr,
            const uint8_t *z_img_ptr,
            struct Buffer *constants);
