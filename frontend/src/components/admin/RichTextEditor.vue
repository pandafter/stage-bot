<script setup lang="ts">
import { useEditor, EditorContent } from '@tiptap/vue-3'
import StarterKit from '@tiptap/starter-kit'
import Link from '@tiptap/extension-link'
import { watch } from 'vue'

const props = defineProps<{ modelValue: string }>()
const emit = defineEmits<{ 'update:modelValue': [value: string] }>()

const editor = useEditor({
  content: props.modelValue,
  extensions: [
    StarterKit,
    Link.configure({ openOnClick: false }),
  ],
  onUpdate: ({ editor }) => {
    emit('update:modelValue', editor.getHTML())
  },
})

watch(() => props.modelValue, (val) => {
  if (editor.value && editor.value.getHTML() !== val) {
    editor.value.commands.setContent(val, { emitUpdate: false })
  }
})
</script>

<template>
  <div class="border rounded-lg overflow-hidden">
    <div v-if="editor" class="flex gap-1 p-2 border-b bg-gray-50">
      <button type="button" @click="editor.chain().focus().toggleBold().run()"
        :class="{ 'bg-gray-200': editor.isActive('bold') }"
        class="px-2 py-1 rounded text-sm font-bold hover:bg-gray-200">B</button>
      <button type="button" @click="editor.chain().focus().toggleItalic().run()"
        :class="{ 'bg-gray-200': editor.isActive('italic') }"
        class="px-2 py-1 rounded text-sm italic hover:bg-gray-200">I</button>
      <button type="button" @click="editor.chain().focus().toggleBulletList().run()"
        :class="{ 'bg-gray-200': editor.isActive('bulletList') }"
        class="px-2 py-1 rounded text-sm hover:bg-gray-200">• Lista</button>
    </div>
    <EditorContent :editor="editor" class="p-3 min-h-[100px] prose prose-sm max-w-none" />
  </div>
</template>
