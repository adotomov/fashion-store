// Renders one or more JSON-LD blocks into the DOM. Used for structured data
// that depends on client-fetched data (e.g. a product), which therefore can't
// go through a route's server-rendered meta(). Googlebot executes JS and picks
// these up on render; static structured data should prefer meta() instead.
export function JsonLd({ data }: { data: Record<string, unknown> | Record<string, unknown>[] }) {
  const blocks = Array.isArray(data) ? data : [data];
  return (
    <>
      {blocks.map((block, i) => (
        <script
          // eslint-disable-next-line react/no-array-index-key
          key={i}
          type="application/ld+json"
          dangerouslySetInnerHTML={{ __html: JSON.stringify(block) }}
        />
      ))}
    </>
  );
}
