package timezone

import "time"

// Indonesia tidak memakai jam musim panas, jadi offset tetap selalu benar, dan tidak bergantung pada
// data zona waktu yang sering tidak ada di image Docker berukuran kecil.
var WIB = time.FixedZone("WIB", 7*60*60)
