package ecache

import "golang.org/x/image/math/fixed"

import "github.com/memmaker/battleground/engine/etxt/emask"

// A GlyphCacheHandler acts as an intermediator between a glyph cache
// and another object, typically a Renderer, to give the later a clear
// target interface to conform to while abstracting the details of an
// underlying cache, which might be finickier to deal with directly
// in a performant way.
//
// GlyphCacheHandler's can't be used concurrently unless the concrete
// implementation explicitly says otherwise.
type GlyphCacheHandler interface {

    // --- configuration notification methods ---
    // Update methods (called only if required so overhead can be low).
    // Passed values must always be non-nil, except for NotifyOtherChange.

    // Notifies that the font in use has changed.
    NotifyFontChange(*Font)

    // Notifies that the text size (in pixels) has changed.
    NotifySizeChange(fixed.Int26_6)

    // Notifies that the rasterizer has changed. Typically, the
    // rasterizer's CacheSignature() will be used to tell them apart.
    NotifyRasterizerChange(emask.Rasterizer) // called on config changes too

    // Notifies that the fractional drawing position has changed.
    // Only the _6 decimal bits must be considered.
    NotifyFractChange(fixed.Point26_6)

    // TODO: I have the suspicion that something is not working properly
    //       with NotifyFractChange ^, as if quantization is not being
    //       hanlded properly in caches.

    //NotifyOtherChange(any) // more methods like this could be added

    // --- cache access methods ---

    // Gets the mask image for the given glyph index and current configuration.
    // The bool indicates whether the mask has been found (as it may be nil).
    GetMask(GlyphIndex) (GlyphMask, bool)

    // Passes a mask image for the given glyph index and current
    // configuration to the underlying cache. PassMask should only
    // be called after GetMask() fails.
    //
    // Given a specific configuration, the contents of the mask image
    // must always be consistent. This implies that passed masks may be
    // ignored if a mask is already cached under that configuration, as
    // it will be considered superfluous. In other words: passing different
    // masks for the same configuration may cause inconsistent results.
    PassMask(GlyphIndex, GlyphMask)

    // Notice that many more methods could be provided, like Get/Pass
    // for Advance, Kern, Bounds, etc., and other methods like Clear()
    // or ReleaseFont(), but since etxt doesn't need that, the interface
    // is limited to masks. You can expand whatever you want with your
    // own interfaces and type assertions.
    //
    // Hinting is also another interesting topic, but since sfnt doesn't
    // apply hinting instructions, there's not much to do here. Even if sfnt
    // did, managing glyph "variants" would be wiser, as hinting instructions
    // often exist only for a few characters at a few specific sizes only,
    // and you may not want to keep lots of superfluous duplicated masks for
    // hinted and unhinted configs.
}
