{{/*
    All templates prefixed by "entry-" will be automatically registered.

    If you want to know the syntax of template, see
    https://golang.org/pkg/text/template.

    There are predefined functions such as image, bgm, etc.

    Functions:
        image() string

            returns html string of the form `<img src="<src>">` with
            image url assigned to src attribute that is chosen
            randomly.

        image_url() string

            is similar to image(), but returns only image url that is chosen randomly. 

        bgm() string

            is similar to image(), but use bgm urls instead of image urls.

        bgm_url() string

            is similar to image_url(), but use bgm urls instead of image urls.

        NOTE

            image urls comes from file images and bgms url from bgms.
            these two files will be exposed to public later.

*/}}

{{define "entry-1"}}
아오야마
===
{{image}}<br>
{{bgm}}
{{end}}

{{define "entry-2"}}
아오야마 미도리
===
{{image}}<br>
{{bgm}}
{{end}}

{{define "entry-3"}}
아오야마 사랑해
===
{{image}}<br>
<p>《【「『 아오야마 미도리 』」】》</p>
{{bgm}}
{{end}}

{{define "entry-4"}}
미도리
===
{{image}}<br>
아오야마<br>
{{bgm}}
{{end}}

{{define "entry-5"}}
아오야마쨩!
===
{{image}}<br>
{{bgm}}
{{end}}

{{define "entry-6"}}
여신 미도리쨩...
===
{{image}}<br>
<p>《【「『 아오야마 미도리 』」】》</p>
{{bgm}}
{{end}}

{{define "entry-7"}}
아오야마 미도리 사랑해주세요!
===
{{image}}<br>
<p>아오야마 미도리 사랑해주세요!</p>
{{bgm}}
{{end}}

{{define "entry-8"}}
미도리쨩...
===
{{image}}<br>
<p>사랑해...</p>
{{bgm}}
{{end}}

{{define "entry-9"}}
아오야마 미도리 사랑해주세요!
===
{{image}}<br>
<p>미도리쨩!</p>
{{bgm}}
{{end}}

{{define "entry-10"}}
사랑해
===
{{image}}<br>
<p>아오야마 미도리쨩을!</p>
{{bgm}}
{{end}}

{{define "entry-11"}}
아오야마 미도리 이쁘다...
===
{{image}}<br>
<p>민나 사랑해줘!</p>
{{bgm}}
{{end}}

{{define "entry-12"}}
아오야마쨩! 아오야마쨩!
===
{{image}}<br>
<p>아오야마 미도리쨩!</p>
{{bgm}}
{{end}}

{{define "entry-13"}}
아오야마 미도리♡♡
===
{{image}}<br>
<p>♡</p>
{{bgm}}
{{end}}

{{define "entry-14"}}
아오야마 미도리 이뻐요
===
{{image}}<br>
{{bgm}}
{{end}}

{{define "entry-15"}}
아오야마 미도리 귀여워요
===
{{image}}<br>
{{bgm}}
{{end}}

{{define "entry-16"}}
아오야마 미도리가 이쁜 이유!
===
{{image}}<br>
<p>♡♡♡</p>
{{bgm}}
{{end}}

{{define "entry-17"}}
미도리쨩 좋아여...
===
{{image}}<br>
<p>끄응...</p>
{{bgm}}
{{end}}

{{define "entry-18"}}
미도리쨔앙~♡
===
{{image}}<br>
{{bgm}}
{{end}}

{{define "entry-19"}}
좋아해요
===
{{image}}<br>
<p>미도리쨩</p>
{{bgm}}
{{end}}

{{define "entry-20"}}
아오야마 미도리 사랑해주세요
===
{{image}}<br>
<p>아오야마 미도리 사랑해주세요</p>
{{bgm}}
{{end}}
