<script setup lang="ts">
defineProps<{ label?: string }>();
const showModal = ref(false);

function close() {
  showModal.value = false;
}

onMounted(() => {});
</script>

<template>
  <button
    class="relative flex overflow-hidden rounded-lg"
    @click="showModal = true"
  >
    <slot />
    <span
      class="opacity-0 hover:opacity-100 transition duration-300 w-full h-full absolute flex items-center bg-black bg-opacity-50 text-xs uppercase tracking-tight p-2"
    >
      {{ label || "Click to zoom" }}
    </span>
  </button>
  <core-modal v-if="showModal" @close="close">
    <div class="z-20">
      <slot name="fullsize" classes="w-auto h-auto max-h-[80vh] max-w-[80vw]" />
    </div>
  </core-modal>
</template>
