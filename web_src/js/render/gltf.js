// import { ModelViewerElement } from '@google/model-viewer';

export async function initGltfViewer() {
  const gltfviewer = await import(/* webpackChunkName: "@google/model-viewer" */'@google/model-viewer');
}
