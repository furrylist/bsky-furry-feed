<script setup lang="ts">
defineProps<{ disabled?: boolean }>();
const $emit = defineEmits<{
  (event: "comment", comment: string): void;
  (event: "attach", data: Uint8Array): void;
}>();

const attachmentRef = ref<HTMLInputElement>();

const text = ref("");
const showUploadModal = ref(false);

function comment() {
  $emit("comment", text.value);
  text.value = "";
}

async function upload() {
  const file = attachmentRef.value!.files?.item(0);
  if (!file) {
    return;
  }
  showUploadModal.value = false;
  $emit("attach", await file.bytes());
}
</script>

<template>
  <div class="flex gap-3 mt-5 mb-24 text-sm flex-col">
    <textarea
      v-model="text"
      class="flex-1 py-2 px-2 rounded-lg border border-gray-300 dark:border-gray-700 bg-transparent"
      rows="4"
      placeholder="Type your comment..."
      type="text"
      :disabled="disabled"
    />
    <div class="ml-auto flex gap-2">
      <button
        class="px-3 py-1 bg-gray-400 dark:bg-gray-600 rounded-lg hover:bg-gray-500 dark:hover:bg-gray-700 h-min"
        @click="showUploadModal = true"
      >
        Add attachment
      </button>
      <button
        :disabled="text.trim().length === 0 || disabled"
        class="px-3 py-1 bg-blue-400 dark:bg-blue-600 rounded-lg hover:bg-blue-500 dark:hover:bg-blue-700 disabled:bg-blue-300 disabled:dark:bg-blue-500 disabled:cursor-not-allowed h-min"
        @click="comment"
      >
        Comment
      </button>
    </div>
    <core-modal v-if="showUploadModal" @close="showUploadModal = false">
      <shared-card class="dark:text-white z-10 bg-white dark:bg-gray-900">
        <input
          class="mb-2 max-w-[300px]"
          ref="attachmentRef"
          type="file"
          name="audit-attachment"
          id="audit-attachment"
        />
        <div class="flex">
          <button
            class="px-3 py-1 bg-gray-400 dark:bg-gray-600 rounded-lg hover:bg-gray-500 dark:hover:bg-gray-700 h-min"
            @click="showUploadModal = false"
          >
            Cancel
          </button>
          <button
            class="ml-auto px-3 py-1 bg-blue-400 dark:bg-blue-600 rounded-lg hover:bg-blue-500 dark:hover:bg-blue-700 disabled:bg-blue-300 disabled:dark:bg-blue-500 disabled:cursor-not-allowed h-min"
            @click="upload"
          >
            Upload
          </button>
        </div>
      </shared-card>
    </core-modal>
  </div>
</template>
