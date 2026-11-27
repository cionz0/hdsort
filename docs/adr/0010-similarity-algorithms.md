# ADR 0010: Similarity algorithms

- Status: Accepted
- Date: 2026-11-27

## Context

ADR-0003 separates exact duplicates from files that merely look or read alike, and it forbids a delete recommendation on a similar group. The algorithm and the threshold were left open until this ADR. A hash that is not a published function is not allowed.

## Decision

Similarity runs after exact grouping. A file whose content ID is shared with another file in the scan is exact only and is not a member of a similar group.

Two published fingerprints are used. They are not compared with each other.

### Images

JPEG, PNG, and GIF use the average hash published with Johannes Buchner’s ImageHash library (`average_hash` in `imagehash/__init__.py`, <https://github.com/JohannesBuchner/imagehash>).

1. Reduce the image to an 8×8 grid. Each sample is the mean luminance of one tile. Luminance is `0.299 R + 0.587 G + 0.114 B`.
2. Set a bit when that sample is strictly greater than the mean of the 64 samples. The library writes this comparison as `pixels > avg`. The hash is 64 bits. The distance between two hashes is the number of bits that differ, which is the library’s `ImageHash` subtraction.

Two images are in the same similar component when that Hamming distance is at most 10. The library defines the distance and does not publish one cutoff; 10 is the operating point for this program. A one-pixel change on an 8×8 field moves the distance by about one bit, so it stays inside the cutoff.

The decoders are the Go standard library (`image/jpeg`, `image/png`, `image/gif`). A file with one of those extensions that does not decode has no image fingerprint. PDF and Office documents are not fingerprinted until a later ADR cites an extractor.

### Plain text

`.txt` and `.md` use simhash as described by Moses Charikar, “Similarity Estimation Techniques from Rounding Algorithms,” *Proceedings of the 34th Annual ACM Symposium on Theory of Computing* (STOC), 2002, <https://doi.org/10.1145/509907.509965>.

The fingerprint is 64 bits. The features are case-folded words of letters and digits, weighted by how often each word occurs. Each word is mapped to 64 signs with FNV-1a 64 (`hash/fnv`), which is the deterministic stand-in for Charikar’s random hyperplanes: a 1-bit adds the weight, a 0-bit subtracts it. The fingerprint bit is 1 only when the sum is strictly positive.

Two texts are in the same similar component when the Hamming distance is at most 3. At 64 bits, three disagreements is a small angle under Charikar’s estimate. Unrelated word lists land near half the bits and stay outside the cutoff.

### Groups

A similar group is one connected component under the rule above, built with `NewSimilarGroup`. The component has no delete recommendation. The owner can still record an explicit delete decision on a member.

## Consequences

- The scan use case depends on a similarity Strategy. It does not embed the average-hash or simhash arithmetic.
- Byte-identical files stay in the exact group only.
- Changing either threshold or either fingerprint is a new ADR.
