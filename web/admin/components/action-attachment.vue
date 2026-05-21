<script setup lang="ts">
const props = defineProps<{
  attachmentId: bigint;
}>();

const api = await useAPI();

const imageUrl = ref<string>();
const error = ref("");

onMounted(async () => {
  const attachment = await api
    .getAttachment({
      attachmentId: props.attachmentId,
    })
    .catch((e) => {
      error.value = `Failed to download blob: ${e}`;
      return null;
    });

  if (!attachment) {
    return;
  }

  const blob = new Blob([attachment.data.buffer as ArrayBuffer], {
    type: attachment.mimeType,
  });
  imageUrl.value = URL.createObjectURL(blob);
});
</script>

<template>
  <shared-card v-if="error" variant="error">{{ error }}</shared-card>
  <div
    v-else-if="!imageUrl"
    class="loading-flash w-full flex aspect-video rounded-lg"
  ></div>
  <nuxt-link
    v-else="imageUrl"
    :href="imageUrl"
    target="_blank"
    class="block flex-1 max-w-[40%]"
  >
    <img class="rounded-lg" :src="imageUrl" alt="" />
  </nuxt-link>
</template>
