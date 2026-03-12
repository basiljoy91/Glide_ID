'use client'

import { useMemo, useState } from 'react'
import Link from 'next/link'
import { blogCategories, blogPosts } from '@/lib/blogData'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'

export function BlogIndex() {
  const [query, setQuery] = useState('')
  const [activeCategory, setActiveCategory] = useState('All')

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase()
    return blogPosts.filter((post) => {
      const matchesCategory = activeCategory === 'All' || post.category === activeCategory
      const matchesQuery =
        q === '' ||
        post.title.toLowerCase().includes(q) ||
        post.excerpt.toLowerCase().includes(q) ||
        post.category.toLowerCase().includes(q)
      return matchesCategory && matchesQuery
    })
  }, [query, activeCategory])

  return (
    <div className="container mx-auto px-4 py-16 space-y-12">
      <div className="grid gap-6 lg:grid-cols-[1.2fr_0.8fr] lg:items-end">
        <div>
          <h1 className="text-4xl font-display font-semibold tracking-tight">Glide ID Journal</h1>
          <p className="mt-4 text-muted-foreground text-lg">
            Product updates, security research, and field notes from enterprise deployments.
          </p>
        </div>
        <div className="rounded-2xl border bg-background/80 p-4">
          <div className="text-xs uppercase tracking-[0.3em] text-muted-foreground">Search</div>
          <Input
            className="mt-3"
            placeholder="Search articles, security, integrations..."
            value={query}
            onChange={(e) => setQuery(e.target.value)}
          />
        </div>
      </div>

      <div className="flex flex-wrap gap-2">
        {blogCategories.map((cat) => (
          <button
            key={cat}
            className={`rounded-full border px-4 py-2 text-xs font-semibold uppercase tracking-[0.2em] ${
              activeCategory === cat
                ? 'bg-primary text-primary-foreground border-primary'
                : 'bg-background/80 text-muted-foreground'
            }`}
            onClick={() => setActiveCategory(cat)}
          >
            {cat}
          </button>
        ))}
      </div>

      <div className="grid gap-6 lg:grid-cols-[1.2fr_0.8fr]">
        <div className="space-y-6">
          {filtered.map((post) => (
            <div key={post.slug} className="rounded-2xl border bg-background/80 p-6">
              <div className="text-xs uppercase tracking-[0.25em] text-muted-foreground">
                {post.category}
              </div>
              <h2 className="mt-3 text-2xl font-display font-semibold">{post.title}</h2>
              <p className="mt-3 text-sm text-muted-foreground">{post.excerpt}</p>
              <div className="mt-4 flex items-center gap-4 text-xs text-muted-foreground">
                <span>{post.date}</span>
                <span>{post.readTime}</span>
              </div>
              <div className="mt-6">
                <Link href={`/blog/${post.slug}`}>
                  <Button variant="outline">Read article</Button>
                </Link>
              </div>
            </div>
          ))}
          {filtered.length === 0 && (
            <div className="rounded-2xl border bg-background/80 p-6 text-sm text-muted-foreground">
              No posts match your search. Try a different keyword or category.
            </div>
          )}
        </div>
        <div className="space-y-6">
          <div className="rounded-2xl border bg-muted/30 p-6">
            <div className="text-xs uppercase tracking-[0.3em] text-muted-foreground">
              Case files
            </div>
            <h3 className="mt-3 text-xl font-display font-semibold">
              Real-world deployment notes
            </h3>
            <p className="mt-3 text-sm text-muted-foreground">
              Field reports on attendance accuracy, compliance automation, and security hardening.
            </p>
            <div className="mt-6">
              <Button>View case files</Button>
            </div>
          </div>
          <div className="rounded-2xl border bg-background/80 p-6">
            <div className="text-xs uppercase tracking-[0.3em] text-muted-foreground">
              Subscribe
            </div>
            <p className="mt-3 text-sm text-muted-foreground">
              Monthly insights on biometric security, compliance, and HR ops automation.
            </p>
            <div className="mt-4">
              <Input placeholder="Email address" />
              <Button className="mt-3 w-full">Join the list</Button>
            </div>
          </div>
        </div>
      </div>

      <Link href="/">
        <Button variant="outline">Back to home</Button>
      </Link>
    </div>
  )
}
