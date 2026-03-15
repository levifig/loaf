declare module "gray-matter" {
  interface GrayMatterFile {
    data: Record<string, unknown>;
    content: string;
    excerpt?: string;
    orig: string;
  }

  interface GrayMatterOption {
    excerpt?: boolean | ((file: GrayMatterFile) => void);
    engines?: Record<string, unknown>;
  }

  function matter(input: string, options?: GrayMatterOption): GrayMatterFile;

  namespace matter {
    function stringify(content: string, data: Record<string, unknown>): string;
  }

  export = matter;
}
