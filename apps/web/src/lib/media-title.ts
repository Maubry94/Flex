export function mediaTitle(filename: string): string {
  const separatorIndex = Math.max(filename.lastIndexOf('/'), filename.lastIndexOf('\\'))
  const extensionIndex = filename.lastIndexOf('.')
  return extensionIndex > separatorIndex + 1 ? filename.slice(0, extensionIndex) : filename
}
