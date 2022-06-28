import $ from 'jquery';
const {csrfToken} = window.config;

async function uploadFile(file, uploadUrl, dropzone) {
  const formData = new FormData();
  formData.append('file', file, file.name);

  const res = await fetch(uploadUrl, {
    method: 'POST',
    headers: {'X-Csrf-Token': csrfToken},
    body: formData,
  });
  const data = await res.json();
  const upfile = {name: file.name, size: file.size, uuid: data.uuid};
  dropzone.dropzone.emit('addedfile', upfile);
  dropzone.dropzone.emit('thumbnail', upfile, `/attachments/${data.uuid}`);
  dropzone.dropzone.emit('complete', upfile);
  dropzone.dropzone.files.push(upfile);
  return data;
}

export function removeUploadedFileFromEditor(editor, fileUuid) {
  // the raw regexp is: /!\[[^\]]*]\(\/attachments\/{uuid}\)/
  const re = new RegExp(`!\\[[^\\]]*]\\(/attachments/${fileUuid}\\)`);
  editor.value(editor.value().replace(re, '')); // at the moment, we assume the editor is an EasyMDE
  if (editor.element) {
    // when using "simple textarea" mode, the value of the textarea should be replaced too.
    editor.element.value = editor.element.value.replace(re, '');
  }
}

function clipboardPastedImages(e) {
  if (!e.clipboardData) return [];

  const files = [];
  for (const item of e.clipboardData.items || []) {
    if (!item.type || !item.type.startsWith('image/')) continue;
    files.push(item.getAsFile());
  }

  if (files.length) {
    e.preventDefault();
    e.stopPropagation();
  }
  return files;
}


function insertAtCursor(field, value) {
  if (field.selectionStart || field.selectionStart === 0) {
    const startPos = field.selectionStart;
    const endPos = field.selectionEnd;
    field.value = field.value.substring(0, startPos) + value + field.value.substring(endPos, field.value.length);
    field.selectionStart = startPos + value.length;
    field.selectionEnd = startPos + value.length;
  } else {
    field.value += value;
  }
}

function replaceAndKeepCursor(field, oldval, newval) {
  if (field.selectionStart || field.selectionStart === 0) {
    const startPos = field.selectionStart;
    const endPos = field.selectionEnd;
    field.value = field.value.replace(oldval, newval);
    field.selectionStart = startPos + newval.length - oldval.length;
    field.selectionEnd = endPos + newval.length - oldval.length;
  } else {
    field.value = field.value.replace(oldval, newval);
  }
}

export function initCompImagePaste($target) {
  $target.each(function () {
    const dropzone = this.querySelector('.dropzone');
    if (!dropzone) {
      return;
    }
    const uploadUrl = dropzone.getAttribute('data-upload-url');
    const dropzoneFiles = dropzone.querySelector('.files');
    for (const textarea of this.querySelectorAll('textarea')) {
      textarea.addEventListener('paste', async (e) => {
        for (const img of clipboardPastedImages(e)) {
          const name = img.name.slice(0, img.name.lastIndexOf('.'));
          insertAtCursor(textarea, `![${name}]()`);
          const data = await uploadFile(img, uploadUrl, dropzone);
          replaceAndKeepCursor(textarea, `![${name}]()`, `![${name}](/attachments/${data.uuid})`);
          const input = $(`<input id="${data.uuid}" name="files" type="hidden">`).val(data.uuid);
          dropzoneFiles.appendChild(input[0]);
        }
      }, false);
    }
  });
}

export function initEasyMDEImagePaste(easyMDE, dropzone, files) {
  const uploadUrl = dropzone.getAttribute('data-upload-url');
  easyMDE.codemirror.on('paste', async (_, e) => {
    for (const img of clipboardPastedImages(e)) {
      const name = img.name.slice(0, img.name.lastIndexOf('.'));
      const data = await uploadFile(img, uploadUrl, dropzone);
      const pos = easyMDE.codemirror.getCursor();
      easyMDE.codemirror.replaceRange(`![${name}](/attachments/${data.uuid})`, pos);
      const input = $(`<input id="${data.uuid}" name="files" type="hidden">`).val(data.uuid);
      files.append(input);
    }
  });
}
