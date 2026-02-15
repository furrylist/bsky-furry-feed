<script setup lang="ts">
import { useKeyboardShurtcuts } from "~/lib/settings";
import { onDocumentEvent, shouldHandleKeypress } from "~/lib/util";

defineProps<{
  name: string;
}>();
const $emit = defineEmits<{
  (event: "reject", reason: string): void;
  (event: "cancel"): void;
}>();

const reasons = [
  "Not a furry account",
  "Spam",
  "Hateful content",
  "Harassment",
  "Inappropriate sexual behavior",
  "AI-generated images",
];

const selectedReasons = ref(new Set<string>());
const showOther = ref(false);
const other = ref("");
const selectedReasonsText = computed(() =>
  [...selectedReasons.value, other.value]
    .filter(Boolean)
    .sort((a, b) => a.localeCompare(b))
    .join("; ")
);

function handleCheck(reason: string, value: boolean) {
  if (value) {
    selectedReasons.value.add(reason);
  } else {
    selectedReasons.value.delete(reason);
  }
}

onDocumentEvent("keydown", (e: KeyboardEvent) => {
  if (!shouldHandleKeypress(e)) return;
  if (e.key === "Enter") {
    (document.querySelector("#reject") as HTMLButtonElement)?.click();
    return;
  }
  if (e.key === "Escape") {
    $emit("cancel");
    return;
  }
  if (!e.key.match(/^[0-9]$/)) return;
  const index = parseInt(e.key) - 1;
  const reason = reasons[index];
  console.log(reason);
  if (index === reasons.length) {
    showOther.value = true;
    (document.querySelector("#other") as HTMLInputElement)?.focus();
  } else if (reason) {
    handleCheck(reason, !selectedReasons.value.has(reason));
  }
});
</script>

<template>
  <core-modal @close="$emit('cancel')">
    <shared-card class="dark:text-white z-10 bg-white dark:bg-gray-900">
      <h2 class="text-lg font-bold">Reject {{ name }}</h2>
      <p class="text-muted mb-3">
        Please select all applicable reasons for the rejection.
      </p>

      <ul class="mb-3">
        <li
          v-for="(reason, idx) in reasons"
          :key="idx"
          class="flex items-center gap-2 border border-gray-300 dark:border-gray-700 rounded-lg px-3 mb-1.5"
        >
          <input
            :id="`reason-${idx}`"
            type="checkbox"
            :checked="selectedReasons.has(reason)"
            @input="
              handleCheck(reason, ($event.target as HTMLInputElement).checked)
            "
          />
          <label
            class="w-full h-full cursor-pointer py-1.5 flex items-center gap-1"
            :for="`reason-${idx}`"
          >
            <kbd v-if="useKeyboardShurtcuts" class="text-xs">{{ idx + 1 }}</kbd>
            {{ reason }}
          </label>
        </li>
        <li
          class="flex items-center gap-2 border border-gray-300 dark:border-gray-700 rounded-lg px-3 mb-1.5"
        >
          <input id="reason-other" v-model="showOther" type="checkbox" />
          <label
            id="other-label"
            class="w-full h-full cursor-pointer py-1.5 flex items-center gap-1"
            for="reason-other"
          >
            <kbd v-if="useKeyboardShurtcuts" class="text-xs">{{
              reasons.length + 1
            }}</kbd>
            Other
          </label>
        </li>
        <li v-if="showOther">
          <input
            v-model="other"
            id="other"
            type="text"
            aria-labelledby="other-label"
            placeholder="Type a reason..."
            class="rounded-lg w-full py-1 px-3 bg-transparent border border-gray-300 dark:border-gray-700"
          />
        </li>
      </ul>

      <div class="flex justify-between">
        <button
          class="py-1 whitespace-nowrap px-2 text-white bg-gray-500 dark:bg-gray-600 hover:bg-gray-600 dark:hover:bg-gray-700 disabled:bg-gray-400 disabled:dark:bg-gray-500 rounded-lg disabled:cursor-not-allowed"
          @click="$emit('cancel')"
        >
          Cancel
        </button>

        <button
          id="reject"
          class="py-1 px-2 mr-1 bg-red-500 dark:bg-red-600 hover:bg-red-600 dark:hover:bg-red-700 disabled:bg-red-400 disabled:dark:bg-red-500 rounded-lg disabled:cursor-not-allowed"
          :disabled="!selectedReasonsText"
          @click="$emit('reject', selectedReasonsText)"
        >
          Reject
        </button>
      </div>
    </shared-card>
  </core-modal>
</template>
