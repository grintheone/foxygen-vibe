// Render from the original file on every save to avoid cumulative JPEG loss.
export async function rotateImageFile(file, quarterTurns) {
    const turns = ((quarterTurns % 4) + 4) % 4;
    if (turns === 0) return file;

    const url = URL.createObjectURL(file);
    try {
        const image = new Image();
        image.src = url;
        await image.decode();
        const canvas = document.createElement("canvas");
        canvas.width = turns % 2 ? image.naturalHeight : image.naturalWidth;
        canvas.height = turns % 2 ? image.naturalWidth : image.naturalHeight;
        const context = canvas.getContext("2d");
        if (!context) throw new Error("Canvas is unavailable");
        context.translate(canvas.width / 2, canvas.height / 2);
        context.rotate(turns * Math.PI / 2);
        context.drawImage(image, -image.naturalWidth / 2, -image.naturalHeight / 2);
        const type = file.type === "image/jpeg" || /\.jpe?g$/i.test(file.name) ? "image/jpeg" : "image/png";
        const blob = await new Promise((resolve, reject) => {
            canvas.toBlob((result) => result ? resolve(result) : reject(new Error("Image encoding failed")), type, 0.95);
        });
        if (blob.size > 25 * 1024 * 1024) throw new Error("Изображение после поворота превышает 25 МБ.");
        const name = file.name.replace(/\.[^.]+$/, "") + (blob.type === "image/jpeg" ? ".jpg" : ".png");
        return new File([blob], name, { type: blob.type, lastModified: Date.now() });
    } finally {
        URL.revokeObjectURL(url);
    }
}
