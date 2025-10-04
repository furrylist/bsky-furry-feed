<script setup lang="ts">
import DOMPurify from "dompurify";
import { marked } from "marked";

const props = defineProps<{
  markdown: string;
}>();

const renderedMarkdown = computed(() => {
  const rendered = marked.parse(props.markdown, {
    gfm: true,
  });
  return DOMPurify.sanitize(rendered);
});
</script>

<template>
  <!-- eslint-disable-next-line vue/no-v-html -->
  <div class="markdown" v-html="renderedMarkdown" />
</template>

<style scoped>
.markdown :deep(a) {
  @apply underline;
  @apply text-blue-600;
  @apply dark:text-blue-400;
}

.markdown :deep(a:hover) {
  @apply no-underline;
}

.markdown :deep(img),
.markdown :deep(table) {
  @apply hidden;
}

.markdown :deep(blockquote) {
  @apply pl-2 border-l-4 border-gray-400 dark:border-gray-700;
}

.markdown :deep(p:not(:last-child)),
.markdown :deep(blockquote:not(:last-child)) {
  @apply mb-2;
}
</style>
