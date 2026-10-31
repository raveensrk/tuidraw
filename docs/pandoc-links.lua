-- With --file-scope, pandoc turns a link to a chapter into "#<file>", but no
-- element carries that id. Point such links at the chapter's H1 instead.
function Pandoc(doc)
  local titles = {}
  doc:walk({
    Header = function(h)
      local file = h.identifier:match("^(.-)__")
      if h.level == 1 and file and not titles[file] then
        titles[file] = h.identifier
      end
    end,
  })
  return doc:walk({
    Link = function(l)
      local title = titles[l.target:match("^#(.*)$") or ""]
      if title then
        l.target = "#" .. title
        return l
      end
    end,
  })
end
