<template>
  <!-- Hanya untuk di atas kertas: merah stempel di atas paper 5,06:1, di atas void tidak cukup kontras.
       Kapital dan spasi renggang khusus stempel karena meniru cap tinta (SRS 11.4). Teks ditulis biasa
       oleh pemakai, CSS yang membuatnya kapital, supaya pembaca layar tidak mengejanya huruf per huruf. -->
  <span
    class="inline-block -rotate-2 rounded-paper border-[1.5px] border-stamp px-2 py-1 font-heading text-sm uppercase tracking-[0.24em] text-stamp"
  >
    <slot />
  </span>
</template>
