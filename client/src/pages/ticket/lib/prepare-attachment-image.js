export async function prepareAttachmentImage(file) {
    if (file.size > 25 * 1024 * 1024) {
        throw new Error("Размер фотографии не должен превышать 25 МБ.");
    }
    const isHeic = /^image\/hei[cf](?:-sequence)?$/i.test(file.type) || /\.hei[cf]$/i.test(file.name);
    if (!isHeic) return file;

    // Lazy-load the decoder only for iPhone HEIC/HEIF photos. Convert before
    // preview/upload so attachments remain viewable in browsers without HEIC.
    const { heicTo } = await import("heic-to/csp");
    const blob = await heicTo({ blob: file, type: "image/jpeg", quality: 0.95 });
    if (blob.size > 25 * 1024 * 1024) {
        throw new Error("Фотография после преобразования превышает 25 МБ.");
    }
    return new File([blob], file.name.replace(/\.[^.]+$/, "") + ".jpg", {
        type: "image/jpeg",
        lastModified: file.lastModified,
    });
}
