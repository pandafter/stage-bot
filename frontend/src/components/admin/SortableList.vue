<script setup lang="ts">
// @ts-ignore — vuedraggable v4 lacks full TypeScript declarations
import draggable from 'vuedraggable'

const props = defineProps<{
  modelValue: unknown[]
  itemKey?: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: unknown[]]
}>()

function onUpdate(list: unknown[]) {
  emit('update:modelValue', list)
}
</script>

<template>
  <draggable
    :model-value="modelValue"
    @update:model-value="onUpdate"
    :item-key="itemKey || 'id'"
    handle=".drag-handle"
    ghost-class="opacity-50"
  >
    <template #item="{ element, index }">
      <div class="flex items-start gap-2 group">
        <span class="drag-handle cursor-grab mt-2 text-gray-300 group-hover:text-gray-500 select-none">⠿</span>
        <div class="flex-1">
          <slot :element="element" :index="index" />
        </div>
      </div>
    </template>
  </draggable>
</template>
