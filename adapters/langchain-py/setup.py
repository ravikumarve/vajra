from setuptools import setup, find_packages

setup(
    name="vajra-langchain",
    version="0.1.0",
    description="VAJRA ephemeral credential integration for LangChain agents",
    long_description=open("README.md").read(),
    long_description_content_type="text/markdown",
    author="VAJRA",
    url="https://github.com/vajra/vajra",
    packages=find_packages(),
    python_requires=">=3.10",
    install_requires=[
        "langchain>=0.3.0",
    ],
    classifiers=[
        "Development Status :: 3 - Alpha",
        "Intended Audience :: Developers",
        "License :: OSI Approved :: MIT License",
        "Programming Language :: Python :: 3",
        "Topic :: Security",
    ],
)
