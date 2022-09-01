{{- if lower .Hasil | contains "accept" -}}
{{- if .Keterangan -}}
Dear authors,

We are pleased to inform you that your abstract titled "{{.Judul}}" has been accepted for the International Conference on Nuclear Science, Technology, and Application (ICONSTA). In order to submit the full manuscript, you will be required to:

1. The reviewer suggested following revisions to the abstract when turning it into a full manuscript for proceeding: 
{{.Keterangan}}

2. Create your EDAS account through EDAS Login
you will need to fill the form provided there to create an account, if you have not already done so.

3. Click on the "submit paper" button on the top left corner of the page. Search for "ICONSTA" on the conference name. Click on the "+" button on the rightmost column of the table. After writing the title of the paper and the abstract on the given form, click the "edit paper" button on the bottom of the screen. You will find in the next page the button to add additional authors and the button to upload your manuscript.

Please note that the manuscript needs to be uploaded in .pdf format.

3. The word template for AIP proceedings can be found here 
https://aip.scitation.org/pb-assets/files/publications/apc/8x11WordTemplates-1607702598523.zip 

while the latex template is given here
https://www.overleaf.com/latex/templates/aip-conference-proceedings/fpznwrhxkkpp

Please do not hesitate to ask us if there are any questions.

Thank you and best regards,

ICONSTA 2022 Organizing Committee
{{- else -}}
Dear authors,

We are pleased to inform you that your abstract titled "{{.Judul}}" has been accepted for the International Conference on Nuclear Science, Technology, and Application (ICONSTA). In order to submit the full manuscript, you will be required to:

1. Create your EDAS account through EDAS Login
you will need to fill the form provided there to create an account, if you have not already done so.

2. Click on the "submit paper" button on the top left corner of the page. Search for "ICONSTA" on the conference name. Click on the "+" button on the rightmost column of the table. After writing the title of the paper and the abstract on the given form, click the "edit paper" button on the bottom of the screen. You will find in the next page the button to add additional authors and the button to upload your manuscript.

Please note that the manuscript needs to be uploaded in .pdf format.

3. The word template for AIP proceedings can be found here 
https://aip.scitation.org/pb-assets/files/publications/apc/8x11WordTemplates-1607702598523.zip 

while the latex template is given here
https://www.overleaf.com/latex/templates/aip-conference-proceedings/fpznwrhxkkpp

Please do not hesitate to ask us if there are any questions.

Thank you and best regards,

ICONSTA 2022 Organizing Committee
{{- end -}}
{{- else -}}
Dear authors,

We regret to decide that your abstract titled "{{.Judul}}" doesn't meet our standard in part of novelty and knowledge contribution for the International Conference on Nuclear Science, Technology, and Application (ICONSTA). 
Please find below our reviewer finding:
{{.Keterangan}}

We are looking forward your new submission to either revise or submit a new abstract. Please do not hesitate to ask us if there are any questions.

Thank you and best regards,

Chief Editor of ICONSTA 2022
{{end -}}